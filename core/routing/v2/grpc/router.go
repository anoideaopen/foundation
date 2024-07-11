package grpc

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/anoideaopen/foundation/core/routing/v2/grpc/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Routing errors.
var (
	ErrUnsupportedMethod        = errors.New("unsupported method")
	ErrInvalidNumberOfArguments = errors.New("invalid number of arguments")
	ErrInterfaceNotProtoMessage = errors.New("interface is not a proto.Message")
)

// Router routes method calls to contract methods based on gRPC service description.
type Router struct {
	methodHandler    map[string]handler // map[protoreflect.FullName]handler
	methodToFunction map[string]string  // map[protoreflect.FullName]URL
	functionToMethod map[string]string  // map[URL]protoreflect.FullName
}

// NewRouter creates a new grpc.Router instance.
func NewRouter() *Router {
	return &Router{
		methodHandler:    make(map[string]handler),
		methodToFunction: make(map[string]string),
		functionToMethod: make(map[string]string),
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

	for _, methodDesc := range desc.Methods {
		md := sd.Methods().ByName(protoreflect.Name(methodDesc.MethodName))

		methodFullName := string(md.FullName())

		if _, ok := r.methodHandler[methodFullName]; ok {
			panic(fmt.Sprintf("method '%s' is already registered", methodFullName))
		}

		h := handler{
			service:       impl,
			authRequired:  true, // auth required by default
			isTransaction: true, // transaction by default
			methodDesc:    methodDesc,
		}

		if ext, ok := proto.GetExtension(md.Options(), pb.E_MethodType).(pb.MethodType); ok {
			switch ext {
			case pb.MethodType_METHOD_TYPE_INVOKE:
				h.authRequired = false
				h.isTransaction = false
				h.isInvoke = true

			case pb.MethodType_METHOD_TYPE_QUERY:
				h.authRequired = false
				h.isTransaction = false
				h.isQuery = true
			}
		}

		if ext, ok := proto.GetExtension(md.Options(), pb.E_MethodAuth).(pb.MethodAuth); ok {
			switch ext {
			case pb.MethodAuth_METHOD_AUTH_ENABLED:
				h.authRequired = true

			case pb.MethodAuth_METHOD_AUTH_DISABLED:
				h.authRequired = false
			}
		}

		url := FullNameToURL(methodFullName)

		r.methodHandler[methodFullName] = h
		r.methodToFunction[methodFullName] = url
		r.functionToMethod[url] = methodFullName
	}
}

// Check validates the provided arguments for the specified method.
func (r *Router) Check(stub shim.ChaincodeStubInterface, method string, args ...string) error {
	h, ok := r.methodHandler[method]
	if !ok {
		return ErrUnsupportedMethod
	}

	if len(args) != h.argCount() {
		return ErrInvalidNumberOfArguments
	}

	if h.authRequired {
		args = args[1:]
	}

	_, err := h.methodDesc.Handler(
		h.service,
		context.Background(),
		func(req any) error {
			msg, ok := req.(proto.Message)
			if !ok {
				return ErrInterfaceNotProtoMessage
			}

			return protojson.Unmarshal([]byte(args[0]), msg)
		},
		func(
			_ context.Context,
			req any,
			_ *grpc.UnaryServerInfo,
			_ grpc.UnaryHandler,
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
func (r *Router) Invoke(stub shim.ChaincodeStubInterface, method string, args ...string) ([]byte, error) {
	h, ok := r.methodHandler[method]
	if !ok {
		return nil, ErrUnsupportedMethod
	}

	if len(args) != h.argCount() {
		return nil, ErrInvalidNumberOfArguments
	}

	ctx := context.Background()

	if h.authRequired {
		ctx = ContextWithSender(ctx, args[0])
		args = args[1:]
	}

	resp, err := h.methodDesc.Handler(
		h.service,
		ContextWithStub(ctx, stub),
		func(in any) error {
			msg, ok := in.(proto.Message)
			if !ok {
				return ErrInterfaceNotProtoMessage
			}

			return protojson.Unmarshal([]byte(args[0]), msg)
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	protoMsg, ok := resp.(proto.Message)
	if !ok {
		return nil, ErrInterfaceNotProtoMessage
	}

	return protojson.Marshal(protoMsg)
}

// Handlers returns a map of method names to chaincode functions.
func (r *Router) Handlers() map[string]string { // map[method]function
	return r.methodToFunction
}

// Method retrieves the method associated with the specified chaincode function.
func (r *Router) Method(function string) (method string) {
	return r.functionToMethod[function]
}

// Function returns the name of the chaincode function by the specified method.
func (r *Router) Function(method string) (function string) {
	return r.methodToFunction[method]
}

// AuthRequired indicates if the method requires authentication.
func (r *Router) AuthRequired(method string) bool {
	return r.methodHandler[method].authRequired
}

// ArgCount returns the number of arguments the method takes (excluding the receiver).
func (r *Router) ArgCount(method string) int {
	return r.methodHandler[method].argCount()
}

// IsTransaction checks if the method is a transaction type.
func (r *Router) IsTransaction(method string) bool {
	return r.methodHandler[method].isTransaction
}

// IsInvoke checks if the method is an invoke type.
func (r *Router) IsInvoke(method string) bool {
	return r.methodHandler[method].isInvoke
}

// IsQuery checks if the method is a query type.
func (r *Router) IsQuery(method string) bool {
	return r.methodHandler[method].isQuery
}

type handler struct {
	service       any
	authRequired  bool
	isTransaction bool
	isInvoke      bool
	isQuery       bool
	methodDesc    grpc.MethodDesc
}

func (h handler) argCount() int {
	const (
		invalidFunction  = 0
		messageOnly      = 1
		senderAndMessage = 2
	)

	if h.service == nil {
		return invalidFunction
	}

	if !h.authRequired {
		return messageOnly
	}

	return senderAndMessage
}
