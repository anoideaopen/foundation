package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/core/routing"
	"github.com/anoideaopen/foundation/core/stringsx"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var (
	// ErrUnsupportedMethod is returned when a method is not supported by the router.
	ErrUnsupportedMethod = errors.New("unsupported method")

	// ErrInvalidNumberOfArguments is returned when the number of arguments provided is not equal to the number required by the method.
	ErrInvalidNumberOfArguments = errors.New("invalid number of arguments")

	// ErrInputNotProtoMessage is returned when the input is not a proto.Message.
	ErrInputNotProtoMessage = errors.New("input is not a proto.Message")
)

// RouterConfig holds configuration options for the Router.
type RouterConfig struct {
	// Fallback is the router to use if the method is not defined in the contract.
	Fallback routing.Router

	// Use function names instead of contract URLs.
	// Example: addBalanceByAdmin instead of /foundationtoken.FiatService/AddBalanceByAdmin.
	UseNames bool
}

// Router routes method calls to contract methods based on gRPC service description.
type Router struct {
	fallback routing.Router
	useURLs  bool

	methods  map[routing.Function]routing.Method
	handlers map[methodName]handler
}

// NewRouter creates a new grpc.Router instance with the given contract and configuration.
//
// Parameters:
//   - baseContract: The contract instance to route methods for.
//   - cfg: Configuration options for the router.
//
// Returns:
//   - *Router: A new Router instance.
func NewRouter(cfg RouterConfig) *Router {
	var methods map[routing.Function]routing.Method
	if cfg.Fallback != nil {
		methods = cfg.Fallback.Methods()
	} else {
		methods = make(map[routing.Function]routing.Method)
	}

	return &Router{
		fallback: cfg.Fallback,
		useURLs:  !cfg.UseNames,
		methods:  methods,
		handlers: make(map[methodName]handler),
	}
}

// RegisterService registers a service and its implementation to the
// concrete type implementing this interface. It may not be called
// once the server has started serving.
// desc describes the service and its methods and handlers. impl is the
// service implementation which is passed to the method handlers.
func (r *Router) RegisterService(desc *grpc.ServiceDesc, impl any) {
	if len(desc.Streams) > 0 {
		panic("stream methods are not supported")
	}

	sd := FindServiceDescriptor(desc.ServiceName)
	if sd == nil {
		panic(fmt.Sprintf("service '%s' not found", desc.ServiceName))
	}

	for _, method := range desc.Methods {
		md := sd.Methods().ByName(protoreflect.Name(method.MethodName))

		var contractFn string
		if ext, ok := proto.GetExtension(md.Options(), E_ContractFunction).(string); ok && ext != "" {
			contractFn = ext
		} else if r.useURLs {
			// Example:
			// "foundation.token.BalanceService.AddBalanceByAdmin" ->
			// "/foundation.token.BalanceService/AddBalanceByAdmin"
			contractFn = FullNameToURL(string(md.FullName()))
		} else {
			// Example:
			// "AddBalanceByAdmin" ->
			// "addBalanceByAdmin"
			contractFn = stringsx.LowerFirstChar(method.MethodName)
		}

		if _, ok := r.methods[contractFn]; ok {
			panic(fmt.Sprintf("contract function '%s' is already registered", contractFn))
		}

		methodType := routing.MethodTypeTransaction
		if ext, ok := proto.GetExtension(md.Options(), E_MethodType).(MethodType); ok {
			switch ext {
			case MethodType_METHOD_TYPE_TRANSACTION:
				methodType = routing.MethodTypeTransaction

			case MethodType_METHOD_TYPE_INVOKE:
				methodType = routing.MethodTypeInvoke

			case MethodType_METHOD_TYPE_QUERY:
				methodType = routing.MethodTypeQuery
			}
		}

		var requireAuth bool
		switch methodType {
		case routing.MethodTypeTransaction:
			requireAuth = true

		case
			routing.MethodTypeInvoke,
			routing.MethodTypeQuery:
			requireAuth = false
		}

		if ext, ok := proto.GetExtension(md.Options(), E_MethodAuth).(MethodAuth); ok {
			switch ext {
			case MethodAuth_METHOD_AUTH_ENABLED:
				requireAuth = true

			case MethodAuth_METHOD_AUTH_DISABLED:
				requireAuth = false
			}
		}

		numArgs := 1
		if requireAuth {
			numArgs = 2
		}

		cm := routing.Method{
			Type:          methodType,
			ChaincodeFunc: contractFn,
			MethodName:    method.MethodName,
			RequiresAuth:  requireAuth,
			NumArgs:       numArgs,
		}

		r.methods[contractFn] = cm
		r.handlers[method.MethodName] = handler{
			service:          impl,
			contractMethod:   cm,
			methodDesc:       method,
			methodDescriptor: md,
		}
	}
}

// Check validates the provided arguments for the specified method.
// It returns an error if the validation fails.
//
// Parameters:
//   - stub: The ChaincodeStubInterface instance to use for the validation.
//   - method: The name of the method to validate arguments for.
//   - args: The arguments to validate.
//
// Returns:
//   - error: An error if the validation fails.
func (r *Router) Check(stub shim.ChaincodeStubInterface, method string, args ...string) error {
	h, ok := r.handlers[method]
	if !ok {
		if r.fallback != nil {
			return r.fallback.Check(stub, method, args...)
		}

		return ErrUnsupportedMethod
	}

	if len(args) != h.contractMethod.NumArgs {
		return ErrInvalidNumberOfArguments
	}

	if h.contractMethod.RequiresAuth {
		args = args[1:]
	}

	_, err := h.methodDesc.Handler(
		h.service,
		context.Background(),
		func(in any) error {
			msg, ok := in.(proto.Message)
			if !ok {
				return ErrInputNotProtoMessage
			}

			return protojson.Unmarshal([]byte(args[0]), msg)
		},
		func(
			ctx context.Context,
			req any,
			info *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (resp any, err error) {
			if validator, ok := req.(interface{ Validate() error }); ok {
				if err := validator.Validate(); err != nil {
					return resp, err
				}
			}

			return resp, nil
		},
	)

	return err
}

// Invoke calls the specified method with the provided arguments.
// It returns a slice of return values and an error if the invocation fails.
//
// Parameters:
//   - stub: The ChaincodeStubInterface instance to use for the invocation.
//   - method: The name of the method to invoke.
//   - args: The arguments to pass to the method.
//
// Returns:
//   - []byte: A slice of bytes (protojson JSON) representing the return values.
//   - error: An error if the invocation fails.
func (r *Router) Invoke(stub shim.ChaincodeStubInterface, method string, args ...string) ([]byte, error) {
	h, ok := r.handlers[method]
	if !ok {
		if r.fallback != nil {
			return r.fallback.Invoke(stub, method, args...)
		}

		return nil, ErrUnsupportedMethod
	}

	if len(args) != h.contractMethod.NumArgs {
		return nil, ErrInvalidNumberOfArguments
	}

	ctx := context.Background()

	if h.contractMethod.RequiresAuth {
		ctx = ContextWithSender(ctx, args[0])
		args = args[1:]
	}

	resp, err := h.methodDesc.Handler(
		h.service,
		ContextWithStub(ctx, stub),
		func(in any) error {
			msg, ok := in.(proto.Message)
			if !ok {
				return ErrInputNotProtoMessage
			}

			return protojson.Unmarshal([]byte(args[0]), msg)
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	if protoMsg, ok := resp.(proto.Message); ok {
		return protojson.Marshal(protoMsg)
	}

	return nil, ErrInputNotProtoMessage
}

// Methods retrieves a map of all available methods, keyed by their chaincode function names.
//
// Returns:
//   - map[routing.Function]routing.Method: A map of all available methods.
func (r *Router) Methods() map[routing.Function]routing.Method {
	return r.methods
}

// handler defines a handler that contains information about a contract method
// and its corresponding gRPC method description.
type handler struct {
	service          any
	contractMethod   routing.Method
	methodDesc       grpc.MethodDesc
	methodDescriptor protoreflect.MethodDescriptor
}

// methodName represents the name of a method in the contract.
type methodName = string

// FullNameToURL transforms a method name from "package.Service.Method" to "/package.Service/Method"
func FullNameToURL(fullMethodName string) string {
	parts := strings.Split(fullMethodName, ".")
	if len(parts) < 2 {
		return ""
	}

	var (
		methodName  = parts[len(parts)-1]
		serviceName = parts[len(parts)-2]
		packageName = strings.Join(parts[:len(parts)-2], ".")
	)

	return fmt.Sprintf("/%s.%s/%s", packageName, serviceName, methodName)
}

// ServiceAndMethod extracts the service name and method name from a URL.
func ServiceAndMethod(url string) (string, string) {
	if len(url) == 0 || url[0] != '/' {
		return "", ""
	}

	// Split the trimmed URL by '/'
	parts := strings.Split(url[1:], "/")
	if len(parts) != 2 {
		return "", ""
	}

	var (
		serviceName = parts[0]
		methodName  = parts[1]
	)

	return serviceName, methodName
}

// FindServiceDescriptor finds the service descriptor by the given service name.
func FindServiceDescriptor(serviceName string) protoreflect.ServiceDescriptor {
	var sd protoreflect.ServiceDescriptor

	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Services().Len(); i++ {
			sd = fd.Services().Get(i)
			if sd.FullName() == protoreflect.FullName(serviceName) {
				return false
			}
		}
		return true
	})
	if sd == nil || sd.FullName() != protoreflect.FullName(serviceName) {
		return nil
	}

	return sd
}
