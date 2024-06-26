package mock

import (
	"context"
	"encoding/json"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type MockClientConn struct {
	caller *Wallet
	ch     string
}

func NewMockClientConn(ch string) *MockClientConn {
	return &MockClientConn{
		ch: ch,
	}
}

func (m *MockClientConn) SetCaller(caller *Wallet) *MockClientConn {
	m.caller = caller
	return m
}

// Invoke performs a unary RPC and returns after the response is received
// into reply.
func (m *MockClientConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	if m.caller == nil {
		return errors.New("caller not set")
	}

	var rawJSON []byte
	if protoMessage, ok := args.(proto.Message); ok {
		rawJSON, _ = protojson.Marshal(protoMessage)
	} else {
		rawJSON, _ = json.Marshal(args)
	}

	_, resp, _ := m.caller.RawSignedInvoke(m.ch, method, string(rawJSON))

	if resp.Error != "" {
		return errors.New(resp.Error)
	}

	if protoMessage, ok := reply.(proto.Message); ok {
		return protojson.Unmarshal([]byte(resp.Result), protoMessage)
	}

	return json.Unmarshal([]byte(resp.Result), reply)
}

// NewStream begins a streaming RPC.
func (m *MockClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	panic("streaming methods are not supported")
}
