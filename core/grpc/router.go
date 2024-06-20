package grpc

import (
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/core/contract"
	"github.com/anoideaopen/foundation/core/stringsx"
	"google.golang.org/grpc"
)

// ErrUnsupportedMethod is returned when a method is not supported by the router.
var ErrUnsupportedMethod = errors.New("unsupported method")

// RouterConfig holds configuration options for the Router.
type RouterConfig struct {
	Fallback               contract.Router
	UnaryServerInterceptor grpc.UnaryServerInterceptor
}

// Router routes method calls to contract methods based on gRPC service description.
type Router struct {
	fallback               contract.Router
	unaryServerInterceptor grpc.UnaryServerInterceptor

	services      map[string]*grpc.ServiceDesc // service name to service description
	serviceMethod map[string]*grpc.ServiceDesc // service method name to service description

	methods map[contract.Function]grpc.MethodDesc
}

// NewRouter creates a new grpc.Router instance with the given contract and configuration.
//
// Parameters:
//   - baseContract: The contract instance to route methods for.
//   - cfg: Configuration options for the router.
//
// Returns:
//   - *Router: A new Router instance.
//   - error: An error if the router setup fails.
func NewRouter(cfg RouterConfig) (*Router, error) {
	return &Router{
		fallback:               cfg.Fallback,
		unaryServerInterceptor: cfg.UnaryServerInterceptor,
		services:               make(map[string]*grpc.ServiceDesc),
		serviceMethod:          make(map[string]*grpc.ServiceDesc),
		methods:                make(map[contract.Function]grpc.MethodDesc),
	}, nil
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

	fallbackMethods := make(map[contract.Function]contract.Method)
	if r.fallback != nil {
		fallbackMethods = r.fallback.Methods()
	}

	r.services[desc.ServiceName] = desc
	for _, method := range desc.Methods {
		contractFn := stringsx.LowerFirstChar(method.MethodName)

		if _, ok := r.methods[contractFn]; ok {
			panic(fmt.Sprintf("contract function '%s' is already registered", contractFn))
		}

		if _, ok := fallbackMethods[contractFn]; ok {
			panic(fmt.Sprintf("contract function '%s' is already registered in fallback router", contractFn))
		}

		r.serviceMethod[method.MethodName] = desc
		r.methods[contractFn] = method
	}
}

// Check validates the provided arguments for the specified method.
// It returns an error if the validation fails.
//
// Parameters:
//   - method: The name of the method to validate arguments for.
//   - args: The arguments to validate.
//
// Returns:
//   - error: An error if the validation fails.
func (r *Router) Check(method string, args ...string) error {
	if r.fallback != nil {
		return r.fallback.Check(method, args...)
	}

	return ErrUnsupportedMethod
}

// Invoke calls the specified method with the provided arguments.
// It returns a slice of return values and an error if the invocation fails.
//
// Parameters:
//   - method: The name of the method to invoke.
//   - args: The arguments to pass to the method.
//
// Returns:
//   - []byte: A slice of bytes (JSON) representing the return values.
//     If the method returns BytesEncoder, it will be encoded to bytes with EncodeToBytes.
//   - error: An error if the invocation fails.
func (r *Router) Invoke(method string, args ...string) ([]byte, error) {
	if r.fallback != nil {
		return r.fallback.Invoke(method, args...)
	}

	return nil, ErrUnsupportedMethod
}

// Methods retrieves a map of all available methods, keyed by their chaincode function names.
//
// Returns:
//   - map[contract.Function]contract.Method: A map of all available methods.
func (r *Router) Methods() map[contract.Function]contract.Method {
	methods := make(map[contract.Function]contract.Method)
	for fn, method := range r.methods {
		mtype, auth, nargs := r.methodOptions(fn)

		methods[fn] = contract.Method{
			Type:          mtype,
			ChaincodeFunc: fn,
			MethodName:    method.MethodName,
			RequiresAuth:  auth,
			NumArgs:       nargs,
		}
	}

	if r.fallback != nil {
		fallbackMethods := r.fallback.Methods()
		for fn, m := range fallbackMethods {
			methods[fn] = m
		}
	}

	return methods
}

func (r *Router) methodOptions(contractFunction string) (mt contract.MethodType, auth bool, nargs int) {
	mt = contract.MethodTypeTransaction
	auth = true
	nargs = 2

	// serviceDescriptor, ok := r.methods[contractFunction]
	// if !ok {
	// 	return
	// }

	return
}
