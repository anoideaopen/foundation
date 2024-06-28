package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/contract"
	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const ExecuteTasksEvent = "executeTasks"

var ErrTasksNotFound = errors.New("no tasks found")

// ExecuteTasksRequest represents a request to execute a group of tasks.
type ExecuteTasksRequest struct {
	Tasks []Task `json:"tasks"`
}

// Task represents an individual task to be executed.
type Task struct {
	ID     string   `json:"id"`     // ID unique task ID
	Method string   `json:"method"` // Method chaincode function to invoke
	Args   []string `json:"args"`   // Args arguments for the chaincode function
}

// TaskExecutor handles the execution of a group of tasks.
type TaskExecutor struct {
	BatchCacheStub *cachestub.BatchCacheStub
	Chaincode      *Chaincode
	CfgBytes       []byte
	SKI            string
	TracingHandler *telemetry.TracingHandler
}

// NewTaskExecutor initializes a new TaskExecutor.
func NewTaskExecutor(stub shim.ChaincodeStubInterface, cfgBytes []byte, cc *Chaincode, tracingHandler *telemetry.TracingHandler) *TaskExecutor {
	return &TaskExecutor{
		BatchCacheStub: cachestub.NewBatchCacheStub(stub),
		CfgBytes:       cfgBytes,
		Chaincode:      cc,
		TracingHandler: tracingHandler,
	}
}

// TasksExecutorHandler executes multiple sub-transactions (tasks) within a single transaction in Hyperledger Fabric,
// using cached state between tasks to solve the MVCC problem. Each request in the arguments contains its own set of
// arguments for the respective chaincode method calls.
func TasksExecutorHandler(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	cfgBytes []byte,
	args []string,
	cc *Chaincode,
) ([]byte, error) {
	tracingHandler := cc.contract.TracingHandler()
	traceCtx, span := tracingHandler.StartNewSpan(traceCtx, ExecuteTasks)
	defer span.End()

	log := logger.Logger()
	txID := stub.GetTxID()
	span.SetAttributes(attribute.String("tx_id", txID))
	start := time.Now()
	defer func() {
		log.Infof("tasks executor: tx id: %s, elapsed: %s", txID, time.Since(start))
	}()

	if len(args) != 1 {
		err := fmt.Errorf("failed to validate args for transaction %s: expected exactly 1 argument, received %d", txID, len(args))
		return nil, handleTasksError(span, err)
	}

	var executeTaskRequest ExecuteTasksRequest
	if err := json.Unmarshal([]byte(args[0]), &executeTaskRequest); err != nil {
		err = fmt.Errorf("failed to unmarshal argument to ExecuteTasksRequest for transaction %s, argument: %s", txID, args[0])
		return nil, handleTasksError(span, err)
	}

	log.Warningf("tasks executor: tx id: %s, txs: %d", txID, len(executeTaskRequest.Tasks))

	if len(executeTaskRequest.Tasks) == 0 {
		err := fmt.Errorf("failed to validate argument: no tasks found in ExecuteTasksRequest for transaction %s: %w", txID, ErrTasksNotFound)
		return nil, handleTasksError(span, err)
	}

	executor := NewTaskExecutor(stub, cfgBytes, cc, tracingHandler)

	response, event, err := executor.ExecuteTasks(traceCtx, executeTaskRequest.Tasks)
	if err != nil {
		return nil, handleTasksError(span, fmt.Errorf("failed to handle task for transaction %s: %w", txID, err))
	}

	eventData, err := pb.Marshal(event)
	if err != nil {
		return nil, handleTasksError(span, fmt.Errorf("failed to marshal event for transaction %s: %w", txID, err))
	}

	err = stub.SetEvent(ExecuteTasksEvent, eventData)
	if err != nil {
		return nil, handleTasksError(span, fmt.Errorf("failed to set event for transaction %s: %w", txID, err))
	}

	data, err := pb.Marshal(response)
	if err != nil {
		return nil, handleTasksError(span, fmt.Errorf("failed to marshal response for transaction %s: %w", txID, err))
	}

	return data, nil
}

// ExecuteTasks processes a group of tasks, returning a group response and event.
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

// validatedTxSenderMethodAndArgs validates the sender, method, and arguments for a transaction.
func (e *TaskExecutor) validatedTxSenderMethodAndArgs(
	traceCtx telemetry.TraceContext,
	batchCacheStub *cachestub.BatchCacheStub,
	task Task,
) (*proto.Address, contract.Method, []string, error) {
	_, span := e.TracingHandler.StartNewSpan(traceCtx, "TaskExecutor.validatedTxSenderMethodAndArgs")
	defer span.End()

	span.AddEvent("parsing chaincode method")
	method, err := e.Chaincode.Method(task.Method)
	if err != nil {
		err = fmt.Errorf("failed to parse chaincode method '%s' for task %s: %w", task.Method, task.ID, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, contract.Method{}, nil, err
	}

	span.AddEvent("validating and extracting invocation context")
	senderAddress, args, nonce, err := e.Chaincode.validateAndExtractInvocationContext(batchCacheStub, method, task.Args)
	if err != nil {
		err = fmt.Errorf("failed to validate and extract invocation context for task %s: %w", task.ID, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, contract.Method{}, nil, err
	}

	span.AddEvent("validating authorization")
	if !method.RequiresAuth || senderAddress == nil {
		err = fmt.Errorf("failed to validate authorization for task %s: sender address is missing", task.ID)
		span.SetStatus(codes.Error, err.Error())
		return nil, contract.Method{}, nil, err
	}
	argsToValidate := append([]string{senderAddress.AddrString()}, args...)

	span.AddEvent("validating arguments")
	if err = e.Chaincode.Router().Check(method.MethodName, argsToValidate...); err != nil {
		err = fmt.Errorf("failed to validate arguments for task %s: %w", task.ID, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, contract.Method{}, nil, err
	}

	span.AddEvent("validating nonce")
	sender := types.NewSenderFromAddr((*types.Address)(senderAddress))
	err = checkNonce(batchCacheStub, sender, nonce)
	if err != nil {
		err = fmt.Errorf("failed to validate nonce for task %s, nonce %d: %w", task.ID, nonce, err)
		span.SetStatus(codes.Error, err.Error())
		return nil, contract.Method{}, nil, err
	}

	return senderAddress, method, args[:method.NumArgs-1], nil
}

// ExecuteTask processes an individual task, returning a transaction response and event.
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

	log := logger.Logger()
	start := time.Now()
	span.SetAttributes(attribute.String("task_method", task.Method))
	span.SetAttributes(attribute.StringSlice("task_args", task.Args))
	span.SetAttributes(attribute.String("task_id", task.ID))
	defer func() {
		log.Infof("task method %s task %s elapsed time %d ms", task.Method, task.ID, time.Since(start).Milliseconds())
	}()

	txCacheStub := batchCacheStub.NewTxCacheStub(task.ID)

	span.AddEvent("configuring chaincode")
	if err := contract.Configure(e.Chaincode.contract, batchCacheStub, e.CfgBytes); err != nil {
		err = fmt.Errorf("failed to configure chaincode for task %s: %w", task.ID, err)
		span.SetStatus(codes.Error, err.Error())
		return handleTaskError(span, task, err)
	}

	span.AddEvent("validating tx sender method and args")
	senderAddress, method, args, err := e.validatedTxSenderMethodAndArgs(traceCtx, batchCacheStub, task)
	if err != nil {
		err = fmt.Errorf("failed to validate transaction sender, method, and arguments for task %s: %w", task.ID, err)
		return handleTaskError(span, task, err)
	}

	span.AddEvent("calling method")
	response, err := e.Chaincode.InvokeContractMethod(traceCtx, txCacheStub, method, senderAddress, args, cfgBytes)
	if err != nil {
		return handleTaskError(span, task, err)
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
	logger.Logger().Error(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

func handleTaskError(span trace.Span, task Task, err error) (*proto.TxResponse, *proto.BatchTxEvent) {
	logger.Logger().Errorf("%s: %s: %s", task.Method, task.ID, err)
	span.SetStatus(codes.Error, err.Error())

	ee := proto.ResponseError{Error: err.Error()}
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
