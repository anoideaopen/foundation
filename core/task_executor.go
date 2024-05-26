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
	ExecuteTasksEvent = "executeTasks"
)

var ErrTasksNotFound = errors.New("tasks not found")

type (
	ExecuteTaskRequest struct {
		Tasks []Task `json:"tasks"`
	}

	Task struct {
		ID     string   `json:"id"`     // ID unique task id
		Method string   `json:"method"` // Method of the chaincode function to invoke
		Args   []string `json:"args"`   // Args to pass to the chaincode function
	}

	TaskExecutor struct {
		BatchCacheStub *cachestub.BatchCacheStub
		ChainCode      *ChainCode
		CfgBytes       []byte
		SKI            string
		TracingHandler *telemetry.TracingHandler
	}
)

// TaskExecutorHandler allow to execute few sub transaction (task) in single transaction in hlf using cache state between requests to solbe mcc problem
// The arguments of this method contains array of requests for execution each request contain own arguments for call method
func TaskExecutorHandler(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	cfgBytes []byte,
	arguments []string,
	cc *ChainCode,
) ([]byte, error) {
	tracingHandler := cc.contract.TracingHandler()
	traceCtx, span := tracingHandler.StartNewSpan(traceCtx, ExecuteTasks)
	defer span.End()

	logger := Logger()
	txID := stub.GetTxID()
	span.SetAttributes(attribute.String("tx_id", txID))
	start := time.Now()
	defer func() {
		logger.Infof("tx id %s elapsed time %d ms", txID, time.Since(start).Milliseconds())
	}()

	if len(arguments) != 1 {
		err := fmt.Errorf("expected 1 argument, got %d", len(arguments))
		return nil, handleTasksError(span, err)
	}

	var executeTaskRequest ExecuteTaskRequest
	if err := json.Unmarshal([]byte(arguments[0]), &executeTaskRequest); err != nil {
		err = fmt.Errorf("unmarshaling argument to ExecuteTaskRequest %s, argument %s", txID, arguments[0])
		return nil, handleTasksError(span, err)
	}
	if len(executeTaskRequest.Tasks) == 0 {
		err := fmt.Errorf("validating argument ExecuteTaskRequest %s: %w", txID, ErrTasksNotFound)
		return nil, handleTasksError(span, err)
	}

	executor := NewTaskExecutor(stub, cfgBytes, cc, tracingHandler)

	response, event, err := executor.ExecuteTasks(traceCtx, executeTaskRequest.Tasks)
	if err != nil {
		return nil, handleTasksError(span, fmt.Errorf("handling tx id %s: %w", txID, err))
	}

	eventData, err := pb.Marshal(event)
	if err != nil {
		return nil, handleTasksError(span, fmt.Errorf("marshalling event tx id %s: %w", txID, err))
	}

	err = stub.SetEvent(ExecuteTasksEvent, eventData)
	if err != nil {
		return nil, handleTasksError(span, fmt.Errorf("setting event tx id %s: %w", txID, err))
	}

	data, err := pb.Marshal(response)
	if err != nil {
		return nil, handleTasksError(span, fmt.Errorf("marshalling response tx id %s: %w", txID, err))
	}

	return data, nil
}

func NewTaskExecutor(stub shim.ChaincodeStubInterface, cfgBytes []byte, cc *ChainCode, tracingHandler *telemetry.TracingHandler) *TaskExecutor {
	return &TaskExecutor{
		BatchCacheStub: cachestub.NewBatchCacheStub(stub),
		CfgBytes:       cfgBytes,
		ChainCode:      cc,
		TracingHandler: tracingHandler,
	}
}

func (e *TaskExecutor) ExecuteTasks(
	traceCtx telemetry.TraceContext,
	tasks []Task,
) (
	*proto.BatchResponse,
	*proto.BatchEvent,
	error,
) {
	traceCtx, span := e.TracingHandler.StartNewSpan(traceCtx, "TaskExecutor.ExecuteTasks")
	defer span.End()

	batchResponse := &proto.BatchResponse{}
	batchEvent := &proto.BatchEvent{}

	for _, task := range tasks {
		txResponse, txEvent := e.ExecuteTask(traceCtx, task, e.BatchCacheStub, e.CfgBytes)
		batchResponse.TxResponses = append(batchResponse.TxResponses, txResponse)
		batchEvent.Events = append(batchEvent.Events, txEvent)
	}

	if err := e.BatchCacheStub.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit changes using BatchCacheStub: %w", err)
	}

	return batchResponse, batchEvent, nil
}

func (e *TaskExecutor) validatedTxSenderMethodAndArgs(
	traceCtx telemetry.TraceContext,
	batchCacheStub *cachestub.BatchCacheStub,
	task Task,
) (*proto.Address, *Method, []string, error) {
	_, span := e.TracingHandler.StartNewSpan(traceCtx, "TaskExecutor.validatedTxSenderMethodAndArgs")
	defer span.End()

	span.AddEvent("parsing chaincode method")
	method, err := e.ChainCode.Method(task.Method)
	if err != nil {
		err = fmt.Errorf("parsing chaincode method '%s', task id %s: %w", task.Method, task.ID, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}

	span.AddEvent("validating and extracting invocation context")
	senderAddress, args, nonce, err := e.ChainCode.validateAndExtractInvocationContext(batchCacheStub, method, task.Method, task.Args)
	if err != nil {
		err = fmt.Errorf("validating and extracting invocation context, task id %s: %w", task.ID, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}

	span.AddEvent("validating authorization")
	if !method.needsAuth || senderAddress == nil {
		err = fmt.Errorf("validating authorization: sender address is missing for task id %s", task.ID)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}
	argsToValidate := append([]string{senderAddress.AddrString()}, args...)

	span.AddEvent("validating arguments")
	err = reflectx.ValidateArguments(e.ChainCode.contract, method.Name, batchCacheStub, argsToValidate...)
	if err != nil {
		err = fmt.Errorf("validating arguments: task id %s: %w", task.ID, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}

	span.AddEvent("validating nonce")
	sender := types.NewSenderFromAddr((*types.Address)(senderAddress))
	err = checkNonce(batchCacheStub, sender, nonce)
	if err != nil {
		err = fmt.Errorf("validating nonce: task id %s, nonce %d: %w", task.ID, nonce, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}

	return senderAddress, method, args[:method.in], nil
}

func (e *TaskExecutor) ExecuteTask(
	traceCtx telemetry.TraceContext,
	task Task,
	batchCacheStub *cachestub.BatchCacheStub,
	cfgBytes []byte,
) (
	*proto.TxResponse,
	*proto.BatchTxEvent,
) {
	traceCtx, span := e.TracingHandler.StartNewSpan(traceCtx, "TaskExecutor.ExecuteTasks")
	defer span.End()

	logger := Logger()
	start := time.Now()
	span.SetAttributes(attribute.String("task_method", task.Method))
	span.SetAttributes(attribute.StringSlice("task_args", task.Args))
	span.SetAttributes(attribute.String("task_id", task.ID))
	defer func() {
		logger.Infof("task method %s task id %s elapsed time %d ms", task.Method, task.ID, time.Since(start).Milliseconds())
	}()

	txCacheStub := batchCacheStub.NewTxCacheStub(task.ID)

	span.AddEvent("validating tx sender method and args")
	senderAddress, method, args, err := e.validatedTxSenderMethodAndArgs(traceCtx, batchCacheStub, task)
	if err != nil {
		errorMessage := "validating tx sender method and args: " + err.Error()
		return handleTaskError(span, task, errorMessage)
	}

	span.AddEvent("calling method")
	response, err := e.ChainCode.callMethod(traceCtx, txCacheStub, method, senderAddress, args, cfgBytes)
	if err != nil {
		return handleTaskError(span, task, err.Error())
	}

	span.AddEvent("commit")
	writes, events := txCacheStub.Commit()

	sort.Slice(txCacheStub.Accounting, func(i, j int) bool {
		return strings.Compare(txCacheStub.Accounting[i].String(), txCacheStub.Accounting[j].String()) < 0
	})

	span.SetStatus(codes.Ok, "")
	return &proto.TxResponse{
			Id:     []byte(task.ID),
			Method: task.Method,
			Writes: writes,
		},
		&proto.BatchTxEvent{
			Id:         []byte(task.ID),
			Method:     task.Method,
			Accounting: txCacheStub.Accounting,
			Events:     events,
			Result:     response,
		}
}

func handleTasksError(span trace.Span, err error) error {
	logger := Logger()
	logger.Error(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

func handleTaskError(span trace.Span, task Task, errorMessage string) (*proto.TxResponse, *proto.BatchTxEvent) {
	ee := proto.ResponseError{Error: errorMessage}
	span.SetStatus(codes.Error, errorMessage)

	return &proto.TxResponse{
			Id:     []byte(task.ID),
			Method: task.Method,
			Error:  &ee,
		}, &proto.BatchTxEvent{
			Id:     []byte(task.ID),
			Method: task.Method,
			Error:  &ee,
		}
}
