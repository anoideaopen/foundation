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
	Channel        string
	Method         string
	Args           []string
	IsSignedInvoke bool
}

type ExecutorResponse struct {
	TxResponse   *proto.TxResponse
	BatchTxEvent *proto.BatchTxEvent
}

func NewExecutorRequest(ch string, fn string, args []string, isSignedInvoke bool) ExecutorRequest {
	return ExecutorRequest{
		Channel:        ch,
		Method:         fn,
		Args:           args,
		IsSignedInvoke: isSignedInvoke,
	}
}

func (w *Wallet) ExecuteSignedInvoke(ch string, fn string, args ...string) ([]byte, error) {
	resp, err := w.TaskExecutor(NewExecutorRequest(ch, fn, args, true))
	if err != nil {
		return nil, err
	}

	return resp.BatchTxEvent.GetResult(), nil
}

func (w *Wallet) TaskExecutor(r ExecutorRequest) (*ExecutorResponse, error) {
	err := w.verifyIncoming(r.Channel, r.Method)
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
		args, _ = w.sign(r.Method, r.Channel, r.Args...)
	}

	task := core.Task{
		ID:     strconv.FormatInt(rand.Int63(), 10),
		Method: r.Method,
		Args:   args,
	}

	bytes, err := json.Marshal(core.ExecuteTasksRequest{Tasks: []core.Task{task}})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tasks ExecuteTasksRequest: %w", err)
	}

	// do invoke chaincode
	peerResponse, err := w.ledger.doInvokeWithPeerResponse(r.Channel, txIDGen(), core.ExecuteTasks, string(bytes))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke method %s: %w", core.ExecuteTasks, err)
	}

	if peerResponse.GetStatus() != http.StatusOK {
		return nil, fmt.Errorf("failed to invoke method %s, status: '%v', message: '%s'", core.ExecuteTasks, peerResponse.GetStatus(), peerResponse.GetMessage())
	}

	var batchResponse proto.BatchResponse
	err = proto2.Unmarshal(peerResponse.GetPayload(), &batchResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal BatchResponse: %w", err)
	}

	batchTxEvent, err := w.getEventByID(r.Channel, task.ID)
	if err != nil {
		return nil, err
	}

	txResponse, err := getTxResponseByID(&batchResponse, task.ID)
	if err != nil {
		return nil, err
	}

	if responseErr := txResponse.GetError(); responseErr != nil {
		return nil, errors.New(responseErr.GetError())
	}

	return &ExecutorResponse{
		TxResponse:   txResponse,
		BatchTxEvent: batchTxEvent,
	}, nil
}

func (w *Wallet) getEventByID(channel string, id string) (*proto.BatchTxEvent, error) {
	e := <-w.ledger.stubs[channel].ChaincodeEventsChannel
	if e.GetEventName() == core.ExecuteTasksEvent {
		batchEvent := proto.BatchEvent{}
		err := proto2.Unmarshal(e.GetPayload(), &batchEvent)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal BatchEvent: %w", err)
		}
		for _, ev := range batchEvent.GetEvents() {
			if string(ev.GetId()) == id {
				return ev, nil
			}
		}
	}
	return nil, fmt.Errorf("failed to find event %s by id %s", core.ExecuteTasksEvent, id)
}

func getTxResponseByID(
	batchResponse *proto.BatchResponse,
	id string,
) (
	*proto.TxResponse,
	error,
) {
	for _, response := range batchResponse.GetTxResponses() {
		if string(response.GetId()) == id {
			return response, nil
		}
	}
	return nil, fmt.Errorf("failed to find response by id %s", id)
}
