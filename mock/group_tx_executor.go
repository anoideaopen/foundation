package mock

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/proto"
	proto2 "google.golang.org/protobuf/proto"
)

type ExecutorRequest struct {
	Channel        string   `json:"ch"`
	Function       string   `json:"fn"`
	Args           []string `json:"args"`
	IsSignedInvoke bool     `json:"isSignedInvoke"`
}

type ExecutorResponse struct {
	TxResponse   *proto.TxResponse
	RequestEvent *proto.BatchTxEvent
}

func NewExecutorRequest(ch string, fn string, args []string, isSignedInvoke bool) ExecutorRequest {
	return ExecutorRequest{
		Channel:        ch,
		Function:       fn,
		Args:           args,
		IsSignedInvoke: isSignedInvoke,
	}
}

func (w *Wallet) ExecuteSignedInvoke(ch string, fn string, args ...string) ([]byte, error) {
	resp, err := w.GroupTxExecutor(NewExecutorRequest(ch, fn, args, true))
	if err != nil {
		return nil, err
	}

	return resp.RequestEvent.GetResult(), nil
}

func (w *Wallet) GroupTxExecutor(r ExecutorRequest) (*ExecutorResponse, error) {
	err := w.verifyIncoming(r.Channel, r.Function)
	if err != nil {
		return nil, fmt.Errorf("failed to verify incoming args: %w", err)
	}

	// setup creator
	cert, err := hex.DecodeString(batchRobotCert)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string batchRobotCert: %w", err)
	}
	w.ledger.stubs[r.Channel].SetCreator(cert)

	var args []string
	if r.IsSignedInvoke {
		args, _ = w.sign(r.Function, r.Channel, r.Args...)
	}

	txRequest := core.TxRequest{
		RequestID: strconv.FormatInt(rand.Int63(), 10),
		Method:    r.Function,
		Args:      args,
	}

	bytes, err := json.Marshal(core.ExecuteGroupTxRequest{Requests: []core.TxRequest{txRequest}})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal requests ExecuteGroupTxRequest: %w", err)
	}

	// do invoke chaincode
	peerResponse, err := w.ledger.doInvokeWithPeerResponse(r.Channel, txIDGen(), core.ExecuteGroupTx, string(bytes))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke method %s: %w", core.ExecuteGroupTx, err)
	}

	if peerResponse.GetStatus() != http.StatusOK {
		return nil, fmt.Errorf("failed to invoke method %s, status: '%v', message: '%s'", core.ExecuteGroupTx, peerResponse.GetStatus(), peerResponse.GetMessage())
	}

	var batchResponse proto.BatchResponse
	err = proto2.Unmarshal(peerResponse.GetPayload(), &batchResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal BatchResponse: %w", err)
	}

	requestEvent, err := w.getEventByRequestID(r.Channel, txRequest.RequestID)
	if err != nil {
		return nil, err
	}

	txResponse, err := getTxResponseByRequestID(&batchResponse, txRequest.RequestID)
	if err != nil {
		return nil, err
	}

	if responseErr := txResponse.GetError(); responseErr != nil {
		return nil, errors.New(responseErr.GetError())
	}

	return &ExecutorResponse{
		TxResponse:   txResponse,
		RequestEvent: requestEvent,
	}, nil
}

func (w *Wallet) getEventByRequestID(channel string, requestID string) (*proto.BatchTxEvent, error) {
	e := <-w.ledger.stubs[channel].ChaincodeEventsChannel
	if e.GetEventName() == core.ExecuteGroupTxEvent {
		batchEvent := proto.BatchEvent{}
		err := proto2.Unmarshal(e.GetPayload(), &batchEvent)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal BatchEvent: %w", err)
		}
		for _, ev := range batchEvent.GetEvents() {
			if string(ev.GetId()) == requestID {
				return ev, nil
			}
		}
	}
	return nil, fmt.Errorf("failed to find event %s for request %s", core.ExecuteGroupTxEvent, requestID)
}

func getTxResponseByRequestID(
	batchResponse *proto.BatchResponse,
	requestID string,
) (
	*proto.TxResponse,
	error,
) {
	for _, response := range batchResponse.GetTxResponses() {
		if string(response.GetId()) == requestID {
			return response, nil
		}
	}
	return nil, fmt.Errorf("failed to find response of batch request %s", requestID)
}
