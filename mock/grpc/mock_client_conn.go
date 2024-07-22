package grpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	coregrpc "github.com/anoideaopen/foundation/core/routing/grpc"
	corepb "github.com/anoideaopen/foundation/core/routing/grpc/proto"
	"github.com/anoideaopen/foundation/mock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type MockClientConn struct {
	caller *mock.Wallet
	ch     string
}

func NewMockClientConn(ch string) *MockClientConn {
	return &MockClientConn{
		ch: ch,
	}
}

func (m *MockClientConn) SetCaller(caller *mock.Wallet) *MockClientConn {
	m.caller = caller
	return m
}

// Invoke performs a unary RPC and returns after the response is received
// into reply.
func (m *MockClientConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	if m.caller == nil {
		return errors.New("caller not set")
	}

	protoMessage, ok := args.(proto.Message)
	if !ok {
		panic("only proto messages are supported")
	}

	rawJSON, _ := protojson.Marshal(protoMessage)

	serviceName, methodName := coregrpc.URLToServiceAndMethod(method)

	sd := coregrpc.FindServiceDescriptor(serviceName)
	if sd == nil {
		panic("service not found")
	}

	md := sd.Methods().ByName(protoreflect.Name(methodName))
	if md == nil {
		panic("method not found")
	}

	var resp mock.TxResponse
	if ext, ok := proto.GetExtension(md.Options(), corepb.E_MethodType).(corepb.MethodType); ok {
		switch ext {
		case corepb.MethodType_METHOD_TYPE_TRANSACTION:
			_, resp, _ = m.caller.RawSignedInvoke(m.ch, method, string(rawJSON))

		case corepb.MethodType_METHOD_TYPE_QUERY:
			peerResp, err := m.caller.InvokeWithPeerResponse(m.ch, method, string(rawJSON))
			if err != nil {
				return err
			}

			if peerResp.GetStatus() != http.StatusOK {
				return fmt.Errorf(
					"unexpected status code: %d, message: %s",
					peerResp.GetStatus(),
					peerResp.GetMessage(),
				)
			}

			resp.Result = string(peerResp.GetPayload())

		default:
			panic("method type not supported")
		}
	} else {
		_, resp, _ = m.caller.RawSignedInvoke(m.ch, method, string(rawJSON))
	}

	if resp.Error != "" {
		return errors.New(resp.Error)
	}

	protoMessage, ok = reply.(proto.Message)
	if !ok {
		panic("only proto messages are supported")
	}

	return protojson.Unmarshal([]byte(resp.Result), protoMessage)
}

// NewStream begins a streaming RPC.
func (m *MockClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	panic("streaming methods are not supported")
}
