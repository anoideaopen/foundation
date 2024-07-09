package core

import (
	"context"

	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// ChaincodeInvocation holds information about a chaincode invocation.
type ChaincodeInvocation struct {
	Stub  shim.ChaincodeStubInterface
	Trace telemetry.TraceContext
}

// ContextWithChaincodeInvocation adds a ChaincodeInvocation to the context.
func ContextWithChaincodeInvocation(parent context.Context, inv *ChaincodeInvocation) context.Context {
	return context.WithValue(parent, chaincodeInvocationKey, inv)
}

// ChaincodeInvocationFromContext retrieves a ChaincodeInvocation from the context.
func ChaincodeInvocationFromContext(ctx context.Context) *ChaincodeInvocation {
	if inv, ok := ctx.Value(chaincodeInvocationKey).(*ChaincodeInvocation); ok {
		return inv
	}

	return nil
}

// ctxKey is a type used for keys in context values.
type ctxKey string

const chaincodeInvocationKey ctxKey = "chaincodeInvocation"
