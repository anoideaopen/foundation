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
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

const (
	BatcherBatchExecuteEvent                    = "batcherBatchExecute"
	TxBatcherRequestType     BatcherRequestType = "tx"
)

type (
	BatcherRequestType string

	// BatcherRequest represents the data required to execute a Hyperledger Fabric chaincode.
	BatcherRequest struct {
		BatcherRequestID   string             `json:"batcher_request_id"` // BatcherRequestID batcher request id
		Channel            string             `json:"channel"`            // Channel on which the chaincode will be invoked
		Chaincode          string             `json:"chaincode"`          // Name of the chaincode to invoke
		Method             string             `json:"function"`           // Name of the chaincode function to invoke
		Args               []string           `json:"args"`               // Arguments to pass to the chaincode function
		BatcherRequestType BatcherRequestType `json:"batcherRequestType"` // tx, swaps, swaps_keys, multi_swaps, multi_swaps_keys
	}

	BatcherBatchExecuteRequestDTO struct {
		Requests []BatcherRequest `json:"requests"`
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

	var batchDTO BatcherBatchExecuteRequestDTO
	if err := json.Unmarshal([]byte(arguments[0]), &batchDTO); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BatcherBatchExecuteRequestDTO: %w", err)
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

	eventData, err := pb.Marshal(batchEvent)
	if err != nil {
		return nil, fmt.Errorf("failed marshalling batcher event: %w", err)
	}

	if err = stub.SetEvent(BatcherBatchExecuteEvent, eventData); err != nil {
		return nil, fmt.Errorf("failed setting batch event: %w", err)
	}

	responseBytes, err := json.Marshal(batchResponse)
	if err != nil {
		return nil, fmt.Errorf("failed marshalling batch response: %w", err)
	}

	return responseBytes, nil
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
	*proto.BatcherBatchResponse,
	*proto.BatcherBatchEvent,
	error,
) {
	var (
		response = &proto.BatcherBatchResponse{}
		event    = &proto.BatcherBatchEvent{}
	)

	for _, request := range requests {
		var (
			txResponse *proto.BatcherRequestResponse
			txEvent    *proto.BatcherRequestEvent
		)

		switch request.BatcherRequestType {
		case TxBatcherRequestType:
			txResponse, txEvent = b.HandleRequest(traceCtx, request, b.batchCacheStub, b.cfgBytes)
		default:
			txResponse = txResponseWithError(
				txResponse,
				fmt.Errorf("unsupported batcher request type %s request.BatcherRequestID %s", request.BatcherRequestType, request.BatcherRequestID),
			)
		}
		response.RequestResponses = append(response.RequestResponses, txResponse)
		event.Events = append(event.Events, txEvent)
	}

	if err := b.batchCacheStub.Commit(); err != nil {
		return response, event, fmt.Errorf("failed to commit changes using batchCacheStub: %w", err)
	}

	return response, event, nil
}

func (b *Batcher) validatedTxSenderMethodAndArgs(
	stub *cachestub.BatchCacheStub,
	request BatcherRequest,
) (*proto.Address, *Fn, []string, error) {
	method, err := b.cc.methods.Method(request.Method)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parsing method '%s' in tx '%s': %w", request.Method, request.BatcherRequestID, err)
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
		return nil, nil, nil, fmt.Errorf("incorrect tx %s nonce: %w", request.BatcherRequestID, err)
	}
	return senderAddress, method, args, nil
}

func (b *Batcher) HandleRequest(
	traceCtx telemetry.TraceContext,
	request BatcherRequest,
	stub *cachestub.BatchCacheStub,
	cfgBytes []byte,
) (
	*proto.BatcherRequestResponse,
	*proto.BatcherRequestEvent,
) {
	var (
		txCacheStub = stub.NewTxCacheStub(request.BatcherRequestID)
		txResponse  = &proto.BatcherRequestResponse{
			BatcherRequestId: request.BatcherRequestID,
			Method:           request.Method,
		}
		txEvent = &proto.BatcherRequestEvent{
			BatcherRequestId: request.BatcherRequestID,
		}
	)

	if err := b.saveBatchRequestID(request.BatcherRequestID); err != nil {
		return txResponseWithError(txResponse, err), txEvent
	}

	senderAddress, method, args, err := b.validatedTxSenderMethodAndArgs(stub, request)
	if err != nil {
		return txResponseWithError(txResponse, err), txEvent
	}

	txResponse.Result, err = b.cc.callMethod(traceCtx, txCacheStub, method, senderAddress, args, cfgBytes)
	if err != nil {
		return txResponseWithError(txResponse, err), txEvent
	}

	_, txEvent.Events = txCacheStub.Commit()

	sort.Slice(txCacheStub.Accounting, func(i, j int) bool {
		return strings.Compare(txCacheStub.Accounting[i].String(), txCacheStub.Accounting[j].String()) < 0
	})

	txResponse.Accounting = txCacheStub.Accounting

	return txResponse, txEvent
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

func txResponseWithError(
	batchTxEvent *proto.BatcherRequestResponse,
	err error,
) *proto.BatcherRequestResponse {
	responseError := &proto.ResponseError{Error: err.Error()}
	if batchTxEvent != nil {
		batchTxEvent.Error = responseError
	}
	return batchTxEvent
}
