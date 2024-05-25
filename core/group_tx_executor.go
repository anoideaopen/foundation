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
	ExecuteGroupTxEvent = "executeGroupTx"
)

var ErrTxRequestsNotFound = errors.New("requests not found")

type (
	ExecuteGroupTxRequest struct {
		Requests []TxRequest `json:"requests"`
	}

	TxRequest struct {
		RequestID string   `json:"request_id"` // RequestID unique request id
		Method    string   `json:"function"`   // Method of the chaincode function to invoke
		Args      []string `json:"args"`       // Args to pass to the chaincode function
	}

	GroupTxExecutor struct {
		BatchCacheStub *cachestub.BatchCacheStub
		ChainCode      *ChainCode
		CfgBytes       []byte
		SKI            string
		TracingHandler *telemetry.TracingHandler
	}
)

// GroupTxExecutorHandler allow to execute few sub transaction (tx request) in single transaction in hlf using cache state between requests to solbe mcc problem
// The arguments of this method contains array of requests for execution each request contain own arguments for call method
func GroupTxExecutorHandler(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	cfgBytes []byte,
	arguments []string,
	cc *ChainCode,
) ([]byte, error) {
	tracingHandler := cc.contract.TracingHandler()
	traceCtx, span := tracingHandler.StartNewSpan(traceCtx, ExecuteGroupTx)
	defer span.End()

	logger := Logger()
	groupTxID := stub.GetTxID()
	span.SetAttributes(attribute.String("group_tx_id", groupTxID))
	start := time.Now()
	defer func() {
		logger.Infof("group tx %s elapsed time %d ms", groupTxID, time.Since(start).Milliseconds())
	}()

	if len(arguments) != 1 {
		err := fmt.Errorf("expected 1 argument, got %d", len(arguments))
		return nil, groupTxError(span, err)
	}

	var groupTxRequest ExecuteGroupTxRequest
	if err := json.Unmarshal([]byte(arguments[0]), &groupTxRequest); err != nil {
		err = fmt.Errorf("unmarshaling argument to ExecuteGroupTxRequest %s, argument %s", groupTxID, arguments[0])
		return nil, groupTxError(span, err)
	}
	if len(groupTxRequest.Requests) == 0 {
		err := fmt.Errorf("validating argument ExecuteGroupTxRequest %s: %w", groupTxID, ErrTxRequestsNotFound)
		return nil, groupTxError(span, err)
	}

	executor := NewGroupTxExecutor(stub, cfgBytes, cc, tracingHandler)

	response, event, err := executor.ExecuteGroupTx(traceCtx, groupTxRequest.Requests)
	if err != nil {
		return nil, groupTxError(span, fmt.Errorf("handling group tx %s: %w", groupTxID, err))
	}

	eventData, err := pb.Marshal(event)
	if err != nil {
		return nil, groupTxError(span, fmt.Errorf("marshalling group tx event %s: %w", groupTxID, err))
	}

	err = stub.SetEvent(ExecuteGroupTxEvent, eventData)
	if err != nil {
		return nil, groupTxError(span, fmt.Errorf("setting group tx event %s: %w", groupTxID, err))
	}

	data, err := pb.Marshal(response)
	if err != nil {
		return nil, groupTxError(span, fmt.Errorf("marshalling group tx response %s: %w", groupTxID, err))
	}

	return data, nil
}

func NewGroupTxExecutor(stub shim.ChaincodeStubInterface, cfgBytes []byte, cc *ChainCode, tracingHandler *telemetry.TracingHandler) *GroupTxExecutor {
	return &GroupTxExecutor{
		BatchCacheStub: cachestub.NewBatchCacheStub(stub),
		CfgBytes:       cfgBytes,
		ChainCode:      cc,
		TracingHandler: tracingHandler,
	}
}

func (e *GroupTxExecutor) ExecuteGroupTx(
	traceCtx telemetry.TraceContext,
	requests []TxRequest,
) (
	*proto.BatchResponse,
	*proto.BatchEvent,
	error,
) {
	traceCtx, span := e.TracingHandler.StartNewSpan(traceCtx, "GroupTxExecutor.ExecuteGroupTx")
	defer span.End()

	batchResponse := &proto.BatchResponse{}
	batchEvent := &proto.BatchEvent{}

	for _, request := range requests {
		txResponse, txEvent := e.ExecuteTx(traceCtx, request, e.BatchCacheStub, e.CfgBytes)
		batchResponse.TxResponses = append(batchResponse.TxResponses, txResponse)
		batchEvent.Events = append(batchEvent.Events, txEvent)
	}

	if err := e.BatchCacheStub.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit changes using BatchCacheStub: %w", err)
	}

	return batchResponse, batchEvent, nil
}

func (e *GroupTxExecutor) validatedTxSenderMethodAndArgs(
	traceCtx telemetry.TraceContext,
	batchCacheStub *cachestub.BatchCacheStub,
	request TxRequest,
) (*proto.Address, *Method, []string, error) {
	_, span := e.TracingHandler.StartNewSpan(traceCtx, "GroupTxExecutor.validatedTxSenderMethodAndArgs")
	defer span.End()

	span.AddEvent("parsing chaincode method")
	method, err := e.ChainCode.Method(request.Method)
	if err != nil {
		err = fmt.Errorf("parsing chaincode method '%s', request id %s: %w", request.Method, request.RequestID, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}

	span.AddEvent("validating and extracting invocation context")
	senderAddress, args, nonce, err := e.ChainCode.validateAndExtractInvocationContext(batchCacheStub, method, request.Method, request.Args)
	if err != nil {
		err = fmt.Errorf("validating and extracting invocation context, request id %s: %w", request.RequestID, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}

	span.AddEvent("validating authorization")
	if !method.needsAuth || senderAddress == nil {
		err = fmt.Errorf("validating authorization: sender address is missing for request id %s", request.RequestID)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}
	argsToValidate := append([]string{senderAddress.AddrString()}, args...)

	span.AddEvent("validating arguments")
	err = reflectx.ValidateArguments(e.ChainCode.contract, method.Name, batchCacheStub, argsToValidate...)
	if err != nil {
		err = fmt.Errorf("validating arguments: request id %s: %w", request.RequestID, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}

	span.AddEvent("validating nonce")
	sender := types.NewSenderFromAddr((*types.Address)(senderAddress))
	err = checkNonce(batchCacheStub, sender, nonce)
	if err != nil {
		err = fmt.Errorf("validating nonce: request id %s, nonce %d: %w", request.RequestID, nonce, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}

	return senderAddress, method, args[:method.in], nil
}

func (e *GroupTxExecutor) ExecuteTx(
	traceCtx telemetry.TraceContext,
	request TxRequest,
	batchCacheStub *cachestub.BatchCacheStub,
	cfgBytes []byte,
) (
	*proto.TxResponse,
	*proto.BatchTxEvent,
) {
	traceCtx, span := e.TracingHandler.StartNewSpan(traceCtx, "GroupTxExecutor.ExecuteTx")
	defer span.End()

	logger := Logger()
	start := time.Now()
	span.SetAttributes(attribute.String("method", request.Method))
	span.SetAttributes(attribute.StringSlice("args", request.Args))
	span.SetAttributes(attribute.String("request_id", request.RequestID))
	defer func() {
		logger.Infof("request method %s request id %s elapsed time %d ms", request.Method, request.RequestID, time.Since(start).Milliseconds())
	}()

	txCacheStub := batchCacheStub.NewTxCacheStub(request.RequestID)

	span.AddEvent("validating tx sender method and args")
	senderAddress, method, args, err := e.validatedTxSenderMethodAndArgs(traceCtx, batchCacheStub, request)
	if err != nil {
		errorMessage := "validating tx sender method and args: " + err.Error()
		return txRequestError(span, request, errorMessage)
	}

	span.AddEvent("calling method")
	response, err := e.ChainCode.callMethod(traceCtx, txCacheStub, method, senderAddress, args, cfgBytes)
	if err != nil {
		return txRequestError(span, request, err.Error())
	}

	span.AddEvent("commit")
	writes, events := txCacheStub.Commit()

	sort.Slice(txCacheStub.Accounting, func(i, j int) bool {
		return strings.Compare(txCacheStub.Accounting[i].String(), txCacheStub.Accounting[j].String()) < 0
	})

	span.SetStatus(codes.Ok, "")
	return &proto.TxResponse{
			Id:     []byte(request.RequestID),
			Method: request.Method,
			Writes: writes,
		},
		&proto.BatchTxEvent{
			Id:         []byte(request.RequestID),
			Method:     request.Method,
			Accounting: txCacheStub.Accounting,
			Events:     events,
			Result:     response,
		}
}

func groupTxError(span trace.Span, err error) error {
	logger := Logger()
	logger.Error(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

func txRequestError(span trace.Span, request TxRequest, errorMessage string) (*proto.TxResponse, *proto.BatchTxEvent) {
	ee := proto.ResponseError{Error: errorMessage}
	span.SetStatus(codes.Error, errorMessage)

	return &proto.TxResponse{
			Id:     []byte(request.RequestID),
			Method: request.Method,
			Error:  &ee,
		}, &proto.BatchTxEvent{
			Id:     []byte(request.RequestID),
			Method: request.Method,
			Error:  &ee,
		}
}
