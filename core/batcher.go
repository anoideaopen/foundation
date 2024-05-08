package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/hlfcreator"
	"github.com/anoideaopen/foundation/internal/config"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

const (
	BatcherBatchExecuteEvent                    = "batcherBatchExecute"
	TxBatcherRequestType     BatcherRequestType = "tx"
)

type (
	BatcherBatchRequestDTO struct {
		Requests []BatcherRequest `json:"requests"`
	}

	BatcherBatchResponseDTO struct {
		Responses []BatcherResponse `json:"responses"`
	}

	BatcherBatchEventDTO struct {
		Events []BatcherEvent `json:"events"`
	}

	BatcherRequestType string

	// BatcherRequest represents the data required to execute a Hyperledger Fabric chaincode.
	BatcherRequest struct {
		BatcherRequestID   string             `json:"batcher_request_id"`   // BatcherRequestID batcher request id
		Method             string             `json:"method"`               // Method of the chaincode function to invoke
		Args               []string           `json:"args"`                 // Args to pass to the chaincode function
		BatcherRequestType BatcherRequestType `json:"batcher_request_type"` // BatcherRequestType is a condition for choosing how to process a batcher request
	}

	BatcherResponse struct {
		BatcherRequestId string                `json:"batcher_request_id"`
		Error            *BatcherResponseError `json:"error,omitempty"`
		Result           []byte                `json:"result,omitempty"`
		Accounting       []AccountingRecord    `json:"accounting,omitempty"`
	}

	BatcherResponseError struct {
		Code  int32  `json:"code"`
		Error string `json:"error"`
	}

	AccountingRecord struct {
		Token     string `json:"token"`
		Sender    []byte `json:"sender,omitempty"`
		Recipient []byte `json:"recipient,omitempty"`
		Amount    []byte `json:"amount,omitempty"`
		Reason    string `json:"reason"`
	}

	Event struct {
		Name  string `json:"name"`
		Value []byte `json:"value"`
	}

	BatcherEvent struct {
		BatcherRequestId string  `json:"batcher_request_id"`
		Events           []Event `json:"events"`
	}

	Batcher struct {
		batchCacheStub *cachestub.BatchCacheStub
		cc             *ChainCode
		cfgBytes       []byte
		ski            string
	}
)

func BatcherHandler(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	cfgBytes []byte,
	creatorSKI, hashedCert [32]byte,
	arguments []string,
	cc *ChainCode,
) ([]byte, error) {
	if len(arguments) != 1 {
		return nil, fmt.Errorf("expected 1 argument, got %d", len(arguments))
	}

	var batchDTO BatcherBatchRequestDTO
	if err := json.Unmarshal([]byte(arguments[0]), &batchDTO); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BatcherBatchRequestDTO: %w", err)
	}

	batcher, err := NewBatcher(stub, cfgBytes, cc)
	if err != nil {
		return nil, fmt.Errorf("failed to create batchInsertHandler: %w", err)
	}

	if err = batcher.ValidateCreator(creatorSKI, hashedCert); err != nil {
		return nil, fmt.Errorf("failed to validate creator: %w", err)
	}

	batchResponse, batchEvent, err := batcher.HandleBatch(traceCtx, batchDTO.Requests)
	if err != nil {
		return nil, fmt.Errorf("failed handling batch: %w", err)
	}

	batchEventBytes, err := json.Marshal(batchEvent)
	if err != nil {
		return nil, fmt.Errorf("failed marshalling batcher event: %w", err)
	}

	if err = stub.SetEvent(BatcherBatchExecuteEvent, batchEventBytes); err != nil {
		return nil, fmt.Errorf("failed setting batch event: %w", err)
	}

	batchResponseBytes, err := json.Marshal(batchResponse)
	if err != nil {
		return nil, fmt.Errorf("failed marshalling batch response: %w", err)
	}

	return batchResponseBytes, nil
}

func NewBatcher(stub shim.ChaincodeStubInterface, cfgBytes []byte, cc *ChainCode) (Batcher, error) {
	contractCfg, err := config.ContractConfigFromBytes(cfgBytes)
	if err != nil {
		return Batcher{}, fmt.Errorf("LoadConfig: contract config from bytes: %w", err)
	}

	batchCacheStub := cachestub.NewBatchCacheStub(stub)

	return Batcher{
		batchCacheStub: batchCacheStub,
		cfgBytes:       cfgBytes,
		ski:            contractCfg.GetBatcherSKI(),
		cc:             cc,
	}, nil
}

func (b *Batcher) ValidateCreator(creatorSKI [32]byte, hashedCert [32]byte) error {
	skiBytes, err := hex.DecodeString(b.ski)
	if err != nil {
		return fmt.Errorf("failed to decode hex batcherSKI: %w", err)
	}

	err = hlfcreator.ValidateSKI(skiBytes, creatorSKI, hashedCert)
	if err != nil {
		return fmt.Errorf("unauthorized: batcherSKI is not equal creatorSKI and hashedCert: %w", err)
	}

	return nil
}

func (b *Batcher) HandleBatch(
	traceCtx telemetry.TraceContext,
	requests []BatcherRequest,
) (
	*BatcherBatchResponseDTO,
	*BatcherBatchEventDTO,
	error,
) {
	var (
		responseDTO = &BatcherBatchResponseDTO{}
		eventDTO    = &BatcherBatchEventDTO{}
	)

	for _, request := range requests {
		var (
			response BatcherResponse
			event    *BatcherEvent
		)

		switch request.BatcherRequestType {
		case TxBatcherRequestType:
			response, event = b.HandleRequest(traceCtx, request, b.batchCacheStub, b.cfgBytes)
		default:
			msg := fmt.Sprintf("unsupported batcher request type %s batch request %s", request.BatcherRequestType, request.BatcherRequestID)
			response.Error = &BatcherResponseError{
				Code:  404,
				Error: msg,
			}

			response.BatcherRequestId = request.BatcherRequestID
		}
		responseDTO.Responses = append(responseDTO.Responses, response)
		if event != nil {
			eventDTO.Events = append(eventDTO.Events, *event)
		}
	}

	if err := b.batchCacheStub.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit changes using batchCacheStub: %w", err)
	}

	return responseDTO, eventDTO, nil
}

func (b *Batcher) validatedTxSenderMethodAndArgs(
	stub *cachestub.BatchCacheStub,
	request BatcherRequest,
) (*proto.Address, *Fn, []string, error) {
	method, err := b.cc.methods.Method(request.Method)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parsing method '%s' in batch request '%s': %w", request.Method, request.BatcherRequestID, err)
	}

	senderAddress, args, nonce, err := b.cc.validateAndExtractInvocationContext(stub, method, request.Method, request.Args)
	if err != nil {
		return nil, nil, nil, err
	}

	args, err = doPrepareToSave(stub, method, args)
	if err != nil {
		return nil, nil, nil, err
	}

	args = args[:len(method.in)]
	if senderAddress == nil {
		return nil, nil, nil, fmt.Errorf("no sender in batch request %s", request.BatcherRequestID)
	}

	sender := types.NewSenderFromAddr((*types.Address)(senderAddress))
	if err = checkNonce(stub, sender, nonce); err != nil {
		return nil, nil, nil, fmt.Errorf("incorrect batch request %s nonce: %w", request.BatcherRequestID, err)
	}
	return senderAddress, method, args, nil
}

func (b *Batcher) HandleRequest(
	traceCtx telemetry.TraceContext,
	request BatcherRequest,
	stub *cachestub.BatchCacheStub,
	cfgBytes []byte,
) (
	BatcherResponse,
	*BatcherEvent,
) {
	var (
		requestCacheStub = stub.NewTxCacheStub(request.BatcherRequestID)
		response         = BatcherResponse{
			BatcherRequestId: request.BatcherRequestID,
		}
	)

	if err := b.saveBatchRequestID(request.BatcherRequestID); err != nil {
		return batcherResponseWithError(response, err), nil
	}

	senderAddress, method, args, err := b.validatedTxSenderMethodAndArgs(stub, request)
	if err != nil {
		return batcherResponseWithError(response, err), nil
	}

	response.Result, err = b.cc.callMethod(traceCtx, requestCacheStub, method, senderAddress, args, cfgBytes)
	if err != nil {
		return batcherResponseWithError(response, err), nil
	}

	_, events := requestCacheStub.Commit()

	if len(requestCacheStub.Accounting) > 0 {
		sort.Slice(requestCacheStub.Accounting, func(i, j int) bool {
			return strings.Compare(requestCacheStub.Accounting[i].String(), requestCacheStub.Accounting[j].String()) < 0
		})

		response.Accounting = mapAccounting(requestCacheStub.Accounting)
	}

	return response, &BatcherEvent{
		BatcherRequestId: request.BatcherRequestID,
		Events:           mapEvents(events),
	}
}

func mapAccounting(sourceAccounting []*proto.AccountingRecord) []AccountTransaction {
	var targetAccounting []AccountTransaction
	for _, record := range sourceAccounting {
		targetAccounting = append(targetAccounting, AccountTransaction{
			Token:     record.Token,
			Sender:    record.Sender,
			Recipient: record.Recipient,
			Amount:    record.Amount,
			Reason:    record.Reason,
		})
	}
	return targetAccounting
}

func mapEvents(sourceEvents []*proto.Event) []Event {
	var targetEvents []Event
	for _, e := range sourceEvents {
		targetEvents = append(targetEvents, Event{
			Name:  e.Name,
			Value: e.Value,
		})
	}
	return targetEvents
}

func (b *Batcher) saveBatchRequestID(requestID string) error {
	const batcherKeyPrefix = "batcher"

	compositeKey, err := b.batchCacheStub.CreateCompositeKey(batcherKeyPrefix, []string{requestID})
	if err != nil {
		return fmt.Errorf("failed creating composite key: %w", err)
	}

	existing, err := b.batchCacheStub.GetState(compositeKey)
	if err != nil {
		return fmt.Errorf("failed checking if batch request with ID %s has been handled or not", requestID)
	}
	if len(existing) > 0 {
		return fmt.Errorf("request with ID %s has been already handled", requestID)
	}

	if err = b.batchCacheStub.PutState(compositeKey, []byte(requestID)); err != nil {
		return fmt.Errorf("failed saving batch request ID: %w", err)
	}

	return nil
}

func batcherResponseWithError(
	response BatcherResponse,
	err error,
) BatcherResponse {
	response.Error = &BatcherResponseError{Error: err.Error()}
	return response
}
