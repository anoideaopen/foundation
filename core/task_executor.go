package core

import (
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

type job struct {
	i             int
	senderAddress *proto.Address
	method        contract.Method
	args          []string
	nonce         uint64
	err           error
}

// TaskExecutor handles the execution of a group of tasks.
type TaskExecutor struct {
	BatchCacheStub *cachestub.BatchCacheStub
	Chaincode      *Chaincode
	SKI            string
	TracingHandler *telemetry.TracingHandler
}

// NewTaskExecutor initializes a new TaskExecutor.
func NewTaskExecutor(stub shim.ChaincodeStubInterface, cc *Chaincode, tracingHandler *telemetry.TracingHandler) *TaskExecutor {
	return &TaskExecutor{
		BatchCacheStub: cachestub.NewBatchCacheStub(stub),
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

	var executeTaskRequest proto.ExecuteTasksRequest
	if err := pb.Unmarshal([]byte(args[0]), &executeTaskRequest); err != nil {
		err = fmt.Errorf("failed to unmarshal argument to ExecuteTasksRequest for transaction %s, argument: %s", txID, args[0])
		return nil, handleTasksError(span, err)
	}

	log.Warningf("tasks executor: tx id: %s, txs: %d", txID, len(executeTaskRequest.GetTasks()))

	if len(executeTaskRequest.GetTasks()) == 0 {
		err := fmt.Errorf("failed to validate argument: no tasks found in ExecuteTasksRequest for transaction %s: %w", txID, ErrTasksNotFound)
		return nil, handleTasksError(span, err)
	}

	executor := NewTaskExecutor(stub, cc, tracingHandler)

	response, event, err := executor.ExecuteTasks(traceCtx, executeTaskRequest.GetTasks())
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
	tasks []*proto.Task,
) (*proto.BatchResponse, *proto.BatchEvent, error) {
	traceCtx, span := e.TracingHandler.StartNewSpan(traceCtx, "TaskExecutor.ExecuteTasks")
	defer span.End()

	batchResponse := &proto.BatchResponse{}
	batchEvent := &proto.BatchEvent{}

	work1 := make(chan *job, len(tasks))
	work2 := make(chan *job, len(tasks))

	go func() {
		cur := 0
		jobs := make([]*job, len(tasks))
		for j := range work1 {
			jobs[j.i] = j

			for {
				if cur >= len(tasks) || jobs[cur] == nil {
					break
				}

				work2 <- jobs[cur]
				cur++
			}

			if cur >= len(tasks) {
				close(work2)
				break
			}
		}
	}()

	for i := range tasks {
		go func(i int) {
			senderAddress, method, args, nonce, err := e.validatedTxSenderMethodAndArgs(traceCtx, e.BatchCacheStub, tasks[i])
			work1 <- &job{
				i:             i,
				senderAddress: senderAddress,
				method:        method,
				args:          args,
				nonce:         nonce,
				err:           err,
			}
		}(i)
	}

	for j := range work2 {
		txResponse, txEvent := e.ExecuteTask(traceCtx, j, e.BatchCacheStub, tasks[j.i])
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
	task *proto.Task,
) (*proto.Address, contract.Method, []string, uint64, error) {
	_, span := e.TracingHandler.StartNewSpan(traceCtx, "TaskExecutor.validatedTxSenderMethodAndArgs")
	defer span.End()

	span.AddEvent("parsing chaincode method")
	method, err := e.Chaincode.Method(task.GetMethod())
	if err != nil {
		err = fmt.Errorf("failed to parse chaincode method '%s' for task %s: %w", task.GetMethod(), task.GetId(), err)
		span.SetStatus(codes.Error, err.Error())
		return nil, contract.Method{}, nil, 0, err
	}

	span.AddEvent("validating and extracting invocation context")
	senderAddress, args, nonce, err := e.Chaincode.validateAndExtractInvocationContext(batchCacheStub, method, task.GetArgs())
	if err != nil {
		err = fmt.Errorf("failed to validate and extract invocation context for task %s: %w", task.GetId(), err)
		span.SetStatus(codes.Error, err.Error())
		return nil, contract.Method{}, nil, 0, err
	}

	span.AddEvent("validating authorization")
	if !method.RequiresAuth || senderAddress == nil {
		err = fmt.Errorf("failed to validate authorization for task %s: sender address is missing", task.GetId())
		span.SetStatus(codes.Error, err.Error())
		return nil, contract.Method{}, nil, 0, err
	}
	argsToValidate := append([]string{senderAddress.AddrString()}, args...)

	span.AddEvent("validating arguments")
	if err = e.Chaincode.Router().Check(method.MethodName, argsToValidate...); err != nil {
		err = fmt.Errorf("failed to validate arguments for task %s: %w", task.GetId(), err)
		span.SetStatus(codes.Error, err.Error())
		return nil, contract.Method{}, nil, 0, err
	}

	return senderAddress, method, args[:method.NumArgs-1], nonce, nil
}

// ExecuteTask processes an individual task, returning a transaction response and event.
func (e *TaskExecutor) ExecuteTask(
	traceCtx telemetry.TraceContext,
	j *job,
	stub *cachestub.BatchCacheStub,
	task *proto.Task,
) (*proto.TxResponse, *proto.BatchTxEvent) {
	traceCtx, span := e.TracingHandler.StartNewSpan(traceCtx, "TaskExecutor.ExecuteTasks")
	defer span.End()

	log := logger.Logger()
	start := time.Now()
	span.SetAttributes(attribute.String("task_method", task.GetMethod()))
	span.SetAttributes(attribute.StringSlice("task_args", task.GetArgs()))
	span.SetAttributes(attribute.String("task_id", task.GetId()))
	defer func() {
		log.Infof("task method %s task %s elapsed: %s", task.GetMethod(), task.GetId(), time.Since(start))
	}()

	if j.err != nil {
		err := fmt.Errorf("failed to validate for task %s: %w", task.GetId(), j.err)
		return handleTaskError(span, task, err)
	}

	sender := types.NewSenderFromAddr((*types.Address)(j.senderAddress))
	err := checkNonce(e.BatchCacheStub, sender, j.nonce)
	if err != nil {
		err = fmt.Errorf("failed to validate nonce for task %s, nonce %d: %w", task.GetId(), j.nonce, err)
		return handleTaskError(span, task, err)
	}

	txCacheStub := stub.NewTxCacheStub(task.GetId())

	span.AddEvent("calling method")
	response, err := e.Chaincode.InvokeContractMethod(traceCtx, txCacheStub, j.method, j.senderAddress, j.args)
	if err != nil {
		return handleTaskError(span, task, err)
	}

	span.AddEvent("commit")
	writes, events := txCacheStub.Commit()

	sort.Slice(txCacheStub.Accounting, func(i, j int) bool {
		return strings.Compare(txCacheStub.Accounting[i].String(), txCacheStub.Accounting[j].String()) < 0
	})

	span.SetStatus(codes.Ok, "")
	return &proto.TxResponse{Id: []byte(task.GetId()), Method: task.GetMethod(), Writes: writes},
		&proto.BatchTxEvent{
			Id: []byte(task.GetId()), Method: task.GetMethod(),
			Accounting: txCacheStub.Accounting, Events: events, Result: response,
		}
}

func handleTasksError(span trace.Span, err error) error {
	logger.Logger().Error(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

func handleTaskError(span trace.Span, task *proto.Task, err error) (*proto.TxResponse, *proto.BatchTxEvent) {
	logger.Logger().Errorf("%s: %s: %s", task.GetMethod(), task.GetId(), err)
	span.SetStatus(codes.Error, err.Error())

	ee := proto.ResponseError{Error: err.Error()}
	return &proto.TxResponse{Id: []byte(task.GetId()), Method: task.GetMethod(), Error: &ee},
		&proto.BatchTxEvent{Id: []byte(task.GetId()), Method: task.GetMethod(), Error: &ee}
}
