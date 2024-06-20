package grpcctx

import (
	"context"

	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// contextKey is a type used for keys in context values.
type contextKey string

const (
	stubKey   contextKey = "stub"
	senderKey contextKey = "sender"
)

// WithStub adds a stub to the context.
func WithStub(parent context.Context, stub shim.ChaincodeStubInterface) context.Context {
	return context.WithValue(parent, stubKey, stub)
}

// Stub retrieves a stub from the context.
func Stub(parent context.Context) shim.ChaincodeStubInterface {
	stub, ok := parent.Value(stubKey).(shim.ChaincodeStubInterface)
	if !ok {
		return nil
	}

	return stub
}

// WithSender adds a sender to the context.
func WithSender(parent context.Context, sender string) context.Context {
	return context.WithValue(parent, senderKey, sender)
}

// Sender retrieves a sender from the context.
func Sender(parent context.Context) string {
	sender, ok := parent.Value(senderKey).(string)
	if !ok {
		return ""
	}

	return sender
}
