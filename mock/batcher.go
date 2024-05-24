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

func (w *Wallet) BatcherSignedInvoke(ch string, fn string, args ...string) ([]byte, error) {
	_, requestEvent, err := w.BatcherSignedInvokeWithTxEventReturned(ch, fn, args...)
	if err != nil {
		return nil, err
	}

	return requestEvent.GetResult(), nil
}

func (w *Wallet) BatcherSignedInvokeWithTxEventReturned(
	ch string,
	fn string,
	args ...string,
) (
	*proto.TxResponse,
	*proto.BatchTxEvent,
	error,
) {
	err := w.verifyIncoming(ch, fn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify incoming args: %w", err)
	}

	// setup creator
	cert, err := hex.DecodeString(batchRobotCert)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode hex string batchRobotCert: %w", err)
	}
	w.ledger.stubs[ch].SetCreator(cert)

	// sign argument and use output args with signature for invoke chaincode 'batcherBatchExecute'
	argsWithSign, _ := w.sign(fn, ch, args...)

	r := core.BatcherRequest{
		BatcherRequestID: strconv.FormatInt(rand.Int63(), 10),
		Method:           fn,
		Args:             argsWithSign,
	}

	requests := core.ExecuteBatchRequest{Requests: []core.BatcherRequest{r}}
	bytes, err := json.Marshal(requests)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal requests ExecuteBatchRequest: %w", err)
	}

	// do invoke chaincode
	peerResponse, err := w.ledger.doInvokeWithPeerResponse(ch, txIDGen(), core.ExecuteBatch, string(bytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to invoke method %s: %w", core.ExecuteBatch, err)
	}

	if peerResponse.GetStatus() != http.StatusOK {
		return nil, nil, fmt.Errorf("failed to invoke method %s, status: '%v', message: '%s'", core.ExecuteBatch, peerResponse.GetStatus(), peerResponse.GetMessage())
	}

	var batchResponse proto.BatchResponse
	err = proto2.Unmarshal(peerResponse.GetPayload(), &batchResponse)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal BatchResponse: %w", err)
	}

	requestEvent, err := w.getBatcherRequestEventFromChannelByRequestID(ch, r.BatcherRequestID)
	if err != nil {
		return nil, nil, err
	}

	txResponse, err := getTxResponseByRequestID(&batchResponse, r.BatcherRequestID)
	if err != nil {
		return nil, nil, err
	}

	if responseErr := txResponse.GetError(); responseErr != nil {
		return nil, nil, errors.New(responseErr.GetError())
	}

	return txResponse, requestEvent, nil
}

func (w *Wallet) getBatcherRequestEventFromChannelByRequestID(
	channel string,
	requestID string,
) (
	*proto.BatchTxEvent,
	error,
) {
	e := <-w.ledger.stubs[channel].ChaincodeEventsChannel
	if e.GetEventName() == core.ExecuteBatchEvent {
		batchEvent := proto.BatchEvent{}
		err := proto2.Unmarshal(e.GetPayload(), &batchEvent)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal BatcherBatchEvent: %w", err)
		}
		for _, ev := range batchEvent.GetEvents() {
			if string(ev.GetId()) == requestID {
				return ev, nil
			}
		}
	}
	return nil,
		fmt.Errorf(
			"failed to find event %s for request %s",
			core.ExecuteBatchEvent,
			requestID,
		)
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
