package grpc

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

// ContextWithStub adds a stub to the context.
func ContextWithStub(parent context.Context, stub shim.ChaincodeStubInterface) context.Context {
	return context.WithValue(parent, stubKey, stub)
}

// StubFromContext retrieves a stub from the context.
func StubFromContext(parent context.Context) shim.ChaincodeStubInterface {
	stub, ok := parent.Value(stubKey).(shim.ChaincodeStubInterface)
	if !ok {
		return nil
	}

	return stub
}

// ContextWithSender adds a sender to the context.
func ContextWithSender(parent context.Context, sender string) context.Context {
	return context.WithValue(parent, senderKey, sender)
}

// SenderFromContext retrieves a sender from the context.
func SenderFromContext(parent context.Context) string {
	sender, ok := parent.Value(senderKey).(string)
	if !ok {
		return ""
	}

	return sender
}
