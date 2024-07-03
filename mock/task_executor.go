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
	resp, err := w.TaskExecutor(executorRequest)
	if err != nil {
		return nil, fmt.Errorf("execute signed invoke: %wt", err)
	}

	return resp.BatchTxEvent.GetResult(), nil
}

func (w *Wallet) TaskExecutor(r ExecutorRequest) (*ExecutorResponse, error) {
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
	tasks := []*proto.Task{task}
	return w.TasksExecutor(r.Channel, r.Method, tasks)
}

func (w *Wallet) TasksExecutor(channel string, method string, tasks []*proto.Task) (*ExecutorResponse, error) {
	err := w.verifyIncoming(channel, method)
	if err != nil {
		return nil, fmt.Errorf("failed to verify incoming args: %w", err)
	}

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

	var batchResponse proto.BatchResponse
	err = pb.Unmarshal(peerResponse.GetPayload(), &batchResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal BatchResponse: %w", err)
	}

	batchTxEvent, err := w.getEventByID(channel, tasks[0].GetId())
	if err != nil {
		return nil, err
	}

	txResponse, err := getTxResponseByID(&batchResponse, tasks[0].GetId())
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
		err := pb.Unmarshal(e.GetPayload(), &batchEvent)
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
