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
	//nolint:staticcheck
)

func (w *Wallet) BatcherSignedInvoke(ch string, fn string, args ...string) ([]byte, error) {
	response, _, err := w.BatcherSignedInvokeWithTxEventReturned(ch, fn, args...)
	if err != nil {
		return nil, err
	}

	return response.Result, nil
}

func (w *Wallet) BatcherSignedInvokeWithTxEventReturned(
	ch string,
	fn string,
	args ...string,
) (
	*core.BatcherResponse,
	*core.BatcherEvent,
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
		BatcherRequestID:   strconv.FormatInt(rand.Int63(), 10),
		Method:             fn,
		Args:               argsWithSign,
		BatcherRequestType: core.TxBatcherRequestType,
	}

	requests := core.BatcherBatchRequestDTO{Requests: []core.BatcherRequest{r}}
	bytes, err := json.Marshal(requests)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal requests BatcherBatchRequestDTO: %w", err)
	}

	// do invoke chaincode
	peerResponse, err := w.ledger.doInvokeWithPeerResponse(ch, txIDGen(), core.BatcherBatchExecute, string(bytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to invoke method %s: %w", core.BatcherBatchExecute, err)
	}

	if peerResponse.GetStatus() != http.StatusOK {
		return nil, nil, fmt.Errorf("failed to invoke method %s, status: '%v', message: '%s'", core.BatcherBatchExecute, peerResponse.GetStatus(), peerResponse.GetMessage())
	}

	var batchResponse core.BatcherBatchResponseDTO
	err = json.Unmarshal(peerResponse.GetPayload(), &batchResponse)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal BatcherBatchResponseDTO: %w", err)
	}

	requestEvent, err := w.getBatcherRequestEventFromChannelByRequestID(ch, r.BatcherRequestID)
	if err != nil {
		return nil, nil, err
	}

	requestResponse, err := getBatcherRequestResponseByRequestID(&batchResponse, r.BatcherRequestID)
	if err != nil {
		return nil, nil, err
	}

	if responseErr := requestResponse.Error; responseErr != nil {
		return nil, nil, errors.New(responseErr.Error)
	}

	return requestResponse, requestEvent, nil
}

func (w *Wallet) getBatcherRequestEventFromChannelByRequestID(
	channel string,
	requestID string,
) (
	*core.BatcherEvent,
	error,
) {
	e := <-w.ledger.stubs[channel].ChaincodeEventsChannel
	if e.GetEventName() == core.BatcherBatchExecuteEvent {
		batcherBatchEventDTO := core.BatcherBatchEventDTO{}
		err := json.Unmarshal(e.GetPayload(), &batcherBatchEventDTO)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal BatcherBatchEvent: %w", err)
		}
		for _, ev := range batcherBatchEventDTO.Events {
			if ev.BatcherRequestId == requestID {
				return &ev, nil
			}
		}
	}
	return nil,
		fmt.Errorf(
			"failed to find event %s for request %s",
			core.BatcherBatchExecuteEvent,
			requestID,
		)
}

func getBatcherRequestResponseByRequestID(
	batchResponse *core.BatcherBatchResponseDTO,
	requestID string,
) (
	*core.BatcherResponse,
	error,
) {
	for _, response := range batchResponse.Responses {
		if response.BatcherRequestId == requestID {
			return &response, nil
		}
	}
	return nil, fmt.Errorf("failed to find response of batch request %s", requestID)
}
