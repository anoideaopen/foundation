package grpc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/anoideaopen/foundation/core"
	coregrpc "github.com/anoideaopen/foundation/core/routing/grpc"
	corepb "github.com/anoideaopen/foundation/core/routing/grpc/proto"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type MockClientConn struct {
	mockStub  *mockstub.MockStub
	chaincode *core.Chaincode
	user      *mocks.UserFoundation
	t         *testing.T
}

func NewMockClientConn(t *testing.T, mockStub *mockstub.MockStub, chaincode *core.Chaincode) *MockClientConn {
	return &MockClientConn{
		mockStub:  mockStub,
		chaincode: chaincode,
		t:         t,
	}
}

func (m *MockClientConn) SetCaller(user *mocks.UserFoundation) *MockClientConn {
	m.user = user
	return m
}

// Invoke performs a unary RPC and returns after the response is received into reply.
func (m *MockClientConn) Invoke(_ context.Context, method string, args interface{}, reply interface{}, _ ...grpc.CallOption) error {
	if m.user == nil {
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

	var (
		eventError  string
		eventResult string
	)
	if ext, ok := proto.GetExtension(md.Options(), corepb.E_MethodType).(corepb.MethodType); ok {
		switch ext {
		case corepb.MethodType_METHOD_TYPE_TRANSACTION:
			txID, _ := m.mockStub.TxInvokeChaincodeSigned(m.chaincode, method, m.user, "", "", "", string(rawJSON))
			eventResult, eventError = checkEvent(m.t, m.mockStub, txID)

		case corepb.MethodType_METHOD_TYPE_QUERY:
			peerResp := m.mockStub.QueryChaincode(m.chaincode, method, string(rawJSON))
			if peerResp.GetStatus() != http.StatusOK {
				return fmt.Errorf(
					"unexpected status code: %d, message: %s",
					peerResp.GetStatus(),
					peerResp.GetMessage(),
				)
			}
			eventResult = string(peerResp.GetPayload())

		default:
			panic("method type not supported")
		}
	} else {
		txID, _ := m.mockStub.TxInvokeChaincodeSigned(m.chaincode, method, m.user, "", "", "", string(rawJSON))
		eventResult, eventError = checkEvent(m.t, m.mockStub, txID)
	}

	if eventError != "" {
		return errors.New(eventError)
	}

	protoMessage, ok = reply.(proto.Message)
	if !ok {
		panic("only proto messages are supported")
	}

	return protojson.Unmarshal([]byte(eventResult), protoMessage)
}

// NewStream begins a streaming RPC.
func (m *MockClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	panic("streaming methods are not supported")
}

func checkEvent(t *testing.T, mockStub *mockstub.MockStub, txID string) (eventResult string, eventError string) {
	eventName, payload := mockStub.SetEventArgsForCall(0)
	require.Equal(t, core.BatchExecute, eventName)
	events := &pbfound.BatchEvent{}
	require.NoError(t, proto.Unmarshal(payload, events))
	for _, ev := range events.GetEvents() {
		if hex.EncodeToString(ev.GetId()) == txID {
			if ev.GetError() != nil {
				eventError = ev.GetError().GetError()
			}
			eventResult = string(ev.GetResult())
		}
	}
	return
}
