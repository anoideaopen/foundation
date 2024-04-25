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
	pb "github.com/golang/protobuf/proto"
)

func (w *Wallet) BatcherSignedInvoke(ch string, fn string, args ...string) ([]byte, error) {
	err := w.verifyIncoming(ch, fn)
	if err != nil {
		return nil, fmt.Errorf("failed to verify incoming args: %w", err)
	}

	// setup creator
	cert, err := hex.DecodeString(batchRobotCert)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string batchRobotCert: %w", err)
	}
	w.ledger.stubs[ch].SetCreator(cert)

	// sign argument and use output args with signature for invoke chaincode 'batcherBatchExecute'
	argsWithSign, _ := w.sign(fn, ch, args...)

	r := core.BatcherRequest{}
	r.BatcherRequestID = strconv.FormatInt(rand.Int63(), 10)
	r.Chaincode = ch
	r.Method = fn
	r.Args = argsWithSign
	r.BatcherRequestType = core.TxBatcherRequestType

	requests := core.BatcherBatchExecuteRequestDTO{Requests: []core.BatcherRequest{r}}
	bytes, err := json.Marshal(requests)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal requests BatcherBatchExecuteRequestDTO: %w", err)
	}

	// do invoke chaincode
	txID := txIDGen()
	peerResponse, err := w.ledger.doInvokeWithPeerResponse(ch, txID, core.BatcherBatchExecute, string(bytes))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke method %s: %w", core.BatcherBatchExecute, err)
	}

	if peerResponse.Status != http.StatusOK {
		return nil, fmt.Errorf("failed to invoke method %s, status: '%v', message: '%s'", core.BatcherBatchExecute, peerResponse.Status, peerResponse.Message)
	}

	var batchResponse proto.BatcherBatchResponse
	err = json.Unmarshal(peerResponse.Payload, &batchResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal BatcherBatchExecuteResponseDTO: %w", err)
	}

	if len(batchResponse.BatcherTxResponses) != 1 {
		return nil, fmt.Errorf("failed to handle response, current response len is %d", len(batchResponse.BatcherTxResponses))
	}

	e := <-w.ledger.stubs[ch].ChaincodeEventsChannel
	if e.EventName == core.BatcherBatchExecuteEvent {
		events := &proto.BatcherBatchEvent{}
		err = pb.Unmarshal(e.Payload, events)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal BatcherBatchEvent: %w", err)
		}
		for _, ev := range events.BatchTxEvents {
			if ev.BatcherRequestId == r.BatcherRequestID {
				if ev.Error != nil {
					return nil, errors.New(ev.Error.Error)
				}
				return ev.Result, nil
			}
		}
	}

	return nil, fmt.Errorf(
		"failed to find event %s with BatcherRequestId %s",
		core.BatcherBatchExecuteEvent,
		r.BatcherRequestID,
	)
}
