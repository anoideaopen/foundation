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

	var batchDTO BatcherBatchDTO
	if err := json.Unmarshal([]byte(arguments[0]), &batchDTO); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BatcherBatchDTO: %w", err)
	}

	b, err := NewBatchHandler(stub, cfgBytes, cc)
	if err != nil {
		return nil, fmt.Errorf("failed to create batchInsertHandler: %w", err)
	}

	if err := b.ValidateCreator(creatorSKI, hashedCert); err != nil {
		return nil, fmt.Errorf("failed to validate creator: %w", err)
	}

	resp, err := b.HandleBatcherRequests(traceCtx, batchDTO.Requests)
	if err != nil {
		return nil, fmt.Errorf("failed to execute: %w", err)
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return bytes, nil
}

type BatchHandler struct {
	batchCacheStub *cachestub.BatchCacheStub
	cc             *ChainCode
	cfgBytes       []byte
	batcherSKI     string
}

func NewBatchHandler(stub shim.ChaincodeStubInterface, cfgBytes []byte, cc *ChainCode) (BatchHandler, error) {
	contractCfg, err := config.ContractConfigFromBytes(cfgBytes)
	if err != nil {
		return BatchHandler{}, fmt.Errorf("LoadConfig: contract config from bytes: %w", err)
	}

	batchCacheStub := cachestub.NewBatchCacheStub(stub)

	return BatchHandler{
		batchCacheStub: batchCacheStub,
		cfgBytes:       cfgBytes,
		batcherSKI:     contractCfg.BatcherSKI,
		cc:             cc,
	}, nil
}

func (b *BatchHandler) ValidateCreator(creatorSKI [32]byte, hashedCert [32]byte) error {
	batcherSKIBytes, err := hex.DecodeString(b.batcherSKI)
	if err != nil {
		return fmt.Errorf("failed to decode hex batcherSKI: %w", err)
	}

	err = hlfcreator.ValidateSKI(batcherSKIBytes, creatorSKI, hashedCert)
	if err != nil {
		return fmt.Errorf("unauthorized: batcherSKI is not equal creatorSKI and hashedCert: %w", err)
	}

	return nil
}

func (b *BatchHandler) HandleBatcherRequests(traceCtx telemetry.TraceContext, batcherRequests []BatcherRequest) (BatcherInsertResponseDTO, error) {
	var txResponses []*proto.BatcherTxResponse
	var batchTxEvents []*proto.BatcherTxEvent
	for _, request := range batcherRequests {
		switch request.BatcherRequestType {
		case TxBatcherRequestType:
			writes, accounting, events, result, err := b.HandleTxBatcherRequest(traceCtx, request, b.batchCacheStub, b.cfgBytes)

			txResponse := &proto.BatcherTxResponse{
				BatcherRequestId: request.BatcherRequestID,
				Method:           request.Method,
			}
			batchTxEvent := &proto.BatcherTxEvent{
				BatcherRequestId: request.BatcherRequestID,
				Method:           request.Method,
			}
			if err != nil {
				responseError := &proto.ResponseError{Error: err.Error()}
				txResponse.Error = responseError
				batchTxEvent.Error = responseError
			} else {
				txResponse.Writes = writes
				batchTxEvent.Accounting = accounting
				batchTxEvent.Result = result
				batchTxEvent.Events = events
			}
			txResponses = append(txResponses, txResponse)
			batchTxEvents = append(batchTxEvents, batchTxEvent)
		default:
			err := fmt.Errorf("unsupported batcher request type %s request.BatcherRequestID %s", request.BatcherRequestType, request.BatcherRequestID)
			responseError := &proto.ResponseError{Error: err.Error()}
			txResponse := &proto.BatcherTxResponse{
				BatcherRequestId: request.BatcherRequestID,
				Method:           request.Method,
				Error:            responseError,
			}
			batchTxEvent := &proto.BatcherTxEvent{
				BatcherRequestId: request.BatcherRequestID,
				Method:           request.Method,
				Error:            responseError,
			}
			txResponses = append(txResponses, txResponse)
			batchTxEvents = append(batchTxEvents, batchTxEvent)
		}
	}
	if err := b.batchCacheStub.Commit(); err != nil {
		return BatcherInsertResponseDTO{}, fmt.Errorf("failed to commit changes using batchCacheStub: %w", err)
	}

	return BatcherInsertResponseDTO{
		TxResponses: txResponses,
	}, nil
}

type BatcherRequestType string

const (
	TxBatcherRequestType BatcherRequestType = "tx"
)

// BatcherRequest represents the data required to execute a Hyperledger Fabric chaincode.
type BatcherRequest struct {
	BatcherRequestID   string             `json:"batcher_request_id"` // BatcherRequestID batcher request id
	Channel            string             `json:"channel"`            // Channel on which the chaincode will be invoked
	Chaincode          string             `json:"chaincode"`          // Name of the chaincode to invoke
	Method             string             `json:"function"`           // Name of the chaincode function to invoke
	Args               []string           `json:"args"`               // Arguments to pass to the chaincode function
	BatcherRequestType BatcherRequestType `json:"batcherRequestType"` // tx, swaps, swaps_keys, multi_swaps, multi_swaps_keys
}

type BatcherBatchDTO struct {
	Requests []BatcherRequest `json:"requests"`
}

type BatcherInsertResponseDTO struct {
	TxResponses []*proto.BatcherTxResponse
}

func (b *BatchHandler) HandleTxBatcherRequest(
	traceCtx telemetry.TraceContext,
	request BatcherRequest,
	stub *cachestub.BatchCacheStub,
	cfgBytes []byte,
) ([]*proto.WriteElement, []*proto.AccountingRecord, []*proto.Event, []byte, error) {
	txCacheStub := stub.NewTxCacheStub(request.BatcherRequestID)

	method, err := b.cc.methods.Method(request.Method)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parsing method '%s' in tx '%s': %s", request.Method, request.BatcherRequestID, err.Error())
	}

	senderAddress, args, nonce, err := b.cc.validateAndExtractInvocationContext(stub, method, request.Method, request.Args)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	args, err = doPrepareToSave(stub, method, args)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	args = args[:len(method.in)]

	if senderAddress == nil {
		return nil, nil, nil, nil, fmt.Errorf("no sender in BatcherRequestID %s", request.BatcherRequestID)
	}

	sender := types.NewSenderFromAddr((*types.Address)(senderAddress))
	if err = checkNonce(stub, sender, nonce); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("incorrect tx %s nonce: %s", request.BatcherRequestID, err.Error())
	}

	response, err := b.cc.callMethod(traceCtx, txCacheStub, method, senderAddress, args, cfgBytes)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	writes, events := txCacheStub.Commit()

	sort.Slice(txCacheStub.Accounting, func(i, j int) bool {
		return strings.Compare(txCacheStub.Accounting[i].String(), txCacheStub.Accounting[j].String()) < 0
	})

	return writes, txCacheStub.Accounting, events, response, nil
}
