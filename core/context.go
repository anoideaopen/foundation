package core

import (
	"context"

	"github.com/anoideaopen/foundation/core/routing"
	"github.com/anoideaopen/foundation/core/telemetry"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// ChaincodeInvocation holds information about a chaincode invocation.
type ChaincodeInvocation struct {
	Stub           shim.ChaincodeStubInterface
	Router         routing.Router
	Config         *pb.ContractConfig
	TraceCtx       telemetry.TraceContext
	TracingHandler *telemetry.TracingHandler
}

// ctxKey is a type used for keys in context values.
type ctxKey string

const chaincodeInvocationKey ctxKey = "chaincodeInvocation"

// ContextWithChaincodeInvocation adds a ChaincodeInvocation to the context.
func ContextWithChaincodeInvocation(parent context.Context, inv *ChaincodeInvocation) context.Context {
	return context.WithValue(parent, chaincodeInvocationKey, inv)
}

// ChaincodeInvocationFromContext retrieves a ChaincodeInvocation from the context.
func ChaincodeInvocationFromContext(ctx context.Context) *ChaincodeInvocation {
	inv, ok := ctx.Value(chaincodeInvocationKey).(*ChaincodeInvocation)
	if !ok {
		return nil
	}
	return inv
}
