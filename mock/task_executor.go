package mock

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/proto"
	pb "google.golang.org/protobuf/proto"
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
	executorRequest := NewExecutorRequest(ch, fn, args, true)
	resp, err := w.TaskExecutorRequest(ch, executorRequest)
	if err != nil {
		return nil, fmt.Errorf("execute signed invoke: %v", err)
	}

	if len(resp) != 1 {
		return nil, fmt.Errorf("execute signed invoke failed: expected 1 response, got %d", len(resp))
	}

	return resp[0].BatchTxEvent.GetResult(), nil
}

func (w *Wallet) ExecuteNoSignedInvoke(ch string, fn string, args ...string) ([]byte, error) {
	executorRequest := NewExecutorRequest(ch, fn, args, false)
	resp, err := w.TaskExecutorRequest(ch, executorRequest)
	if err != nil {
		return nil, fmt.Errorf("execute signed invoke: %v", err)
	}

	if len(resp) != 1 {
		return nil, fmt.Errorf("execute signed invoke failed: expected 1 response, got %d", len(resp))
	}

	return resp[0].BatchTxEvent.GetResult(), nil
}

func (w *Wallet) TaskExecutorRequest(channel string, requests ...ExecutorRequest) ([]ExecutorResponse, error) {
	tasks := make([]*proto.Task, len(requests))
	for i, r := range requests {
		if r.Channel != channel {
			return nil, fmt.Errorf("channel does not match to request channel")
		}
		var args []string
		if r.IsSignedInvoke {
			args = w.SignArgs(r.Channel, r.Method, r.Args...)
		} else {
			args = r.Args
		}

		task := &proto.Task{
			Id:     strconv.FormatInt(rand.Int63(), 10),
			Method: r.Method,
			Args:   args,
		}

		tasks[i] = task
	}

	batchResponse, err := w.TasksExecutor(channel, tasks)
	if err != nil {
		return nil, err
	}

	batchEvent, err := w.fetchBatchEvent(channel)
	if err != nil {
		return nil, err
	}
	responseMap := make(map[string]*proto.TxResponse)
	for _, response := range batchResponse.GetTxResponses() {
		responseMap[string(response.GetId())] = response
	}
	executorResponses := make([]ExecutorResponse, 0)
	for _, batchTxEvent := range batchEvent.GetEvents() {
		txResponse, ok := responseMap[string(batchTxEvent.GetId())]
		if !ok {
			return nil, fmt.Errorf("could not find response for event %v", batchTxEvent.GetId())
		}

		if responseErr := txResponse.GetError(); responseErr != nil {
			return nil, errors.New(responseErr.GetError())
		}
		executorResponses = append(executorResponses, ExecutorResponse{
			TxResponse:   txResponse,
			BatchTxEvent: batchTxEvent,
		})
	}

	return executorResponses, nil
}

func (w *Wallet) TasksExecutor(channel string, tasks []*proto.Task) (*proto.BatchResponse, error) {
	// setup creator
	cert, err := hex.DecodeString(batchRobotCert)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string batchRobotCert: %w", err)
	}
	w.ledger.stubs[channel].SetCreator(cert)

	bytes, err := pb.Marshal(&proto.ExecuteTasksRequest{Tasks: tasks})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tasks ExecuteTasksRequest: %w", err)
	}

	// do invoke chaincode
	peerResponse, err := w.ledger.doInvokeWithPeerResponse(channel, txIDGen(), core.ExecuteTasks, string(bytes))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke method %s: %w", core.ExecuteTasks, err)
	}

	if peerResponse.GetStatus() != http.StatusOK {
		return nil, fmt.Errorf("failed to invoke method %s, status: '%v', message: '%s'", core.ExecuteTasks, peerResponse.GetStatus(), peerResponse.GetMessage())
	}

	batchResponse := &proto.BatchResponse{}
	err = pb.Unmarshal(peerResponse.GetPayload(), batchResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal BatchResponse: %w", err)
	}

	return batchResponse, nil
}

func (w *Wallet) fetchBatchEvent(channel string) (*proto.BatchEvent, error) {
	e := <-w.ledger.stubs[channel].ChaincodeEventsChannel
	if e.GetEventName() == core.ExecuteTasksEvent {
		batchEvent := &proto.BatchEvent{}
		err := pb.Unmarshal(e.GetPayload(), batchEvent)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal BatchEvent: %w", err)
		}
		return batchEvent, nil
	}
	return nil, fmt.Errorf("failed to find event %s", core.ExecuteTasksEvent)
}
