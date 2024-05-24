package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/reflectx"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	ExecuteBatchEvent = "executeBatch"
	// BatcherRequestIDCompositeKey is a composite key for store batcherRequestID in hlf
	BatcherRequestIDCompositeKey = "batcherRequestID"
)

var ErrRequestsNotFound = errors.New("requests not found")

type (
	ExecuteBatchRequest struct {
		Requests []BatcherRequest `json:"requests"`
	}

	BatcherRequest struct {
		BatcherRequestID string   `json:"batcher_request_id"` // BatcherRequestID batcher request id
		Method           string   `json:"function"`           // Method of the chaincode function to invoke
		Args             []string `json:"args"`               // Args to pass to the chaincode function
	}

	Batcher struct {
		BatchCacheStub *cachestub.BatchCacheStub
		ChainCode      *ChainCode
		CfgBytes       []byte
		SKI            string
		TracingHandler *telemetry.TracingHandler
	}
)

// BatcherHandler allow to execute few sub transaction (batch request) in single transaction in hlf using cache state between requests to solbe mcc problem
// The arguments of this method contains array of requests for execution each request contain own arguments for call method
func BatcherHandler(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	cfgBytes []byte,
	arguments []string,
	cc *ChainCode,
) ([]byte, error) {
	tracingHandler := cc.contract.TracingHandler()
	traceCtx, span := tracingHandler.StartNewSpan(traceCtx, ExecuteBatch)
	defer span.End()

	logger := Logger()
	batchID := stub.GetTxID()
	span.SetAttributes(attribute.String("batch_tx_id", batchID))
	start := time.Now()
	defer func() {
		logger.Infof("batch %s elapsed time %d ms", batchID, time.Since(start).Milliseconds())
	}()

	if len(arguments) != 1 {
		err := fmt.Errorf("expected 1 argument, got %d", len(arguments))
		return nil, batcherHandlerError(span, err)
	}

	var executeBatchRequest ExecuteBatchRequest
	if err := json.Unmarshal([]byte(arguments[0]), &executeBatchRequest); err != nil {
		err = fmt.Errorf("unmarshaling argument to ExecuteBatchRequest %s, argument %s", batchID, arguments[0])
		return nil, batcherHandlerError(span, err)
	}
	if len(executeBatchRequest.Requests) == 0 {
		err := fmt.Errorf("validating argument ExecuteBatchRequest %s: %w", batchID, ErrRequestsNotFound)
		return nil, batcherHandlerError(span, err)
	}

	batcher := NewBatcher(stub, cfgBytes, cc, tracingHandler)

	batchResponse, batchEvent, err := batcher.HandleBatch(traceCtx, executeBatchRequest.Requests)
	if err != nil {
		return nil, batcherHandlerError(span, fmt.Errorf("handling batch %s: %w", batchID, err))
	}

	eventData, err := pb.Marshal(batchEvent)
	if err != nil {
		return nil, batcherHandlerError(span, fmt.Errorf("marshalling batch event %s: %w", batchID, err))
	}

	err = stub.SetEvent(ExecuteBatchEvent, eventData)
	if err != nil {
		return nil, batcherHandlerError(span, fmt.Errorf("setting batch event %s: %w", batchID, err))
	}

	data, err := pb.Marshal(batchResponse)
	if err != nil {
		return nil, batcherHandlerError(span, fmt.Errorf("marshalling batch response %s: %w", batchID, err))
	}

	return data, nil
}

func NewBatcher(stub shim.ChaincodeStubInterface, cfgBytes []byte, cc *ChainCode, tracingHandler *telemetry.TracingHandler) *Batcher {
	return &Batcher{
		BatchCacheStub: cachestub.NewBatchCacheStub(stub),
		CfgBytes:       cfgBytes,
		ChainCode:      cc,
		TracingHandler: tracingHandler,
	}
}

func (b *Batcher) HandleBatch(
	traceCtx telemetry.TraceContext,
	requests []BatcherRequest,
) (
	*proto.BatchResponse,
	*proto.BatchEvent,
	error,
) {
	traceCtx, span := b.TracingHandler.StartNewSpan(traceCtx, "Batcher.HandleBatch")
	defer span.End()

	batchResponse := &proto.BatchResponse{}
	batchEvent := &proto.BatchEvent{}

	for _, request := range requests {
		txResponse, txEvent := b.HandleTxRequest(traceCtx, request, b.BatchCacheStub, b.CfgBytes)
		batchResponse.TxResponses = append(batchResponse.TxResponses, txResponse)
		batchEvent.Events = append(batchEvent.Events, txEvent)
	}

	if err := b.BatchCacheStub.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit changes using BatchCacheStub: %w", err)
	}

	return batchResponse, batchEvent, nil
}

func (b *Batcher) validatedTxSenderMethodAndArgs(
	traceCtx telemetry.TraceContext,
	batchCacheStub *cachestub.BatchCacheStub,
	request BatcherRequest,
) (*proto.Address, *Method, []string, error) {
	_, span := b.TracingHandler.StartNewSpan(traceCtx, "Batcher.validatedTxSenderMethodAndArgs")
	defer span.End()

	span.AddEvent("parsing method")
	method, err := b.ChainCode.Method(request.Method)
	if err != nil {
		span.SetStatus(codes.Error, "parsing method failed")
		return nil, nil, nil, fmt.Errorf("parsing method '%s', batch request id %s: %w", request.Method, request.BatcherRequestID, err)
	}

	span.AddEvent("validating and extracting invocation context")
	senderAddress, args, nonce, err := b.ChainCode.validateAndExtractInvocationContext(batchCacheStub, method, request.Method, request.Args)
	if err != nil {
		span.SetStatus(codes.Error, "validating and extracting invocation context failed")
		return nil, nil, nil, fmt.Errorf("validating and extracting invocation context, batch request id %s: %w", request.BatcherRequestID, err)
	}

	if !method.needsAuth || senderAddress == nil {
		span.SetStatus(codes.Error, "batch request required auth with senderAddreess failed")
		return nil, nil, nil, fmt.Errorf("batch request required auth with senderAddreess, batch request id %s", request.BatcherRequestID)
	}
	argsToValidate := append([]string{senderAddress.AddrString()}, args...)

	span.AddEvent("validating arguments")
	if err := reflectx.ValidateArguments(b.ChainCode.contract, method.Name, batchCacheStub, argsToValidate...); err != nil {
		span.SetStatus(codes.Error, "validating arguments failed")
		return nil, nil, nil, fmt.Errorf("validating arguments, batch request id %s: %w", request.BatcherRequestID, err)
	}

	span.AddEvent("check nonce")
	sender := types.NewSenderFromAddr((*types.Address)(senderAddress))
	if err = checkNonce(batchCacheStub, sender, nonce); err != nil {
		span.SetStatus(codes.Error, "check nonce failed")
		return nil, nil, nil, fmt.Errorf("check nonce, batch request id %s, nonce %d: %w", request.BatcherRequestID, nonce, err)
	}
	return senderAddress, method, args[:method.in], nil
}

func (b *Batcher) HandleTxRequest(
	traceCtx telemetry.TraceContext,
	request BatcherRequest,
	batchCacheStub *cachestub.BatchCacheStub,
	cfgBytes []byte,
) (
	*proto.TxResponse,
	*proto.BatchTxEvent,
) {
	traceCtx, span := b.TracingHandler.StartNewSpan(traceCtx, "Batcher.HandleTxRequest")
	defer span.End()

	logger := Logger()
	start := time.Now()
	span.SetAttributes(attribute.String("method", request.Method))
	span.SetAttributes(attribute.StringSlice("args", request.Args))
	span.SetAttributes(attribute.String("batcher_request_id", request.BatcherRequestID))
	defer func() {
		logger.Infof("batched method %s BatcherRequestID %s elapsed time %d ms", request.Method, request.BatcherRequestID, time.Since(start).Milliseconds())
	}()

	txCacheStub := batchCacheStub.NewTxCacheStub(request.BatcherRequestID)

	span.AddEvent("saving batch request id")
	err := b.saveBatcherRequestID(request.BatcherRequestID)
	if err != nil {
		errorMessage := "saving batch request id: " + err.Error()
		return handleTxRequestError(span, request, errorMessage)
	}

	span.AddEvent("validating tx sender method and args")
	senderAddress, method, args, err := b.validatedTxSenderMethodAndArgs(traceCtx, batchCacheStub, request)
	if err != nil {
		errorMessage := "validating tx sender method and args: " + err.Error()
		return handleTxRequestError(span, request, errorMessage)
	}

	span.AddEvent("calling method")
	response, err := b.ChainCode.callMethod(traceCtx, txCacheStub, method, senderAddress, args, cfgBytes)
	if err != nil {
		return handleTxRequestError(span, request, err.Error())
	}

	span.AddEvent("commit")
	writes, events := txCacheStub.Commit()

	sort.Slice(txCacheStub.Accounting, func(i, j int) bool {
		return strings.Compare(txCacheStub.Accounting[i].String(), txCacheStub.Accounting[j].String()) < 0
	})

	span.SetStatus(codes.Ok, "")
	return &proto.TxResponse{
			Id:     []byte(request.BatcherRequestID),
			Method: request.Method,
			Writes: writes,
		},
		&proto.BatchTxEvent{
			Id:         []byte(request.BatcherRequestID),
			Method:     request.Method,
			Accounting: txCacheStub.Accounting,
			Events:     events,
			Result:     response,
		}
}

func batcherHandlerError(span trace.Span, err error) error {
	logger := Logger()
	logger.Error(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

func handleTxRequestError(span trace.Span, request BatcherRequest, errorMessage string) (*proto.TxResponse, *proto.BatchTxEvent) {
	ee := proto.ResponseError{Error: errorMessage}
	span.SetStatus(codes.Error, errorMessage)

	return &proto.TxResponse{
			Id:     []byte(request.BatcherRequestID),
			Method: request.Method,
			Error:  &ee,
		}, &proto.BatchTxEvent{
			Id:     []byte(request.BatcherRequestID),
			Method: request.Method,
			Error:  &ee,
		}
}

func (b *Batcher) saveBatcherRequestID(requestID string) error {
	compositeKey, err := b.BatchCacheStub.CreateCompositeKey(BatcherRequestIDCompositeKey, []string{requestID})
	if err != nil {
		return fmt.Errorf("creating composite key batcher request id %s: %w", requestID, err)
	}

	existing, err := b.BatchCacheStub.GetState(compositeKey)
	if err != nil {
		return fmt.Errorf("validating batcher request id %s: %w", requestID, err)
	}
	if len(existing) > 0 {
		return fmt.Errorf("validating batcher request id %s: already exists", requestID)
	}

	err = b.BatchCacheStub.PutState(compositeKey, []byte(requestID))
	if err != nil {
		return fmt.Errorf("saving batch request ID %s: %w", requestID, err)
	}

	return nil
}
