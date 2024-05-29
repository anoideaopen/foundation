package core

import (
	"encoding/hex"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/core/multiswap"
	"github.com/anoideaopen/foundation/core/reflectx"
	"github.com/anoideaopen/foundation/core/swap"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/internal/config"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
)

const robotSideTimeout = 300 // 5 minutes

func (cc *ChainCode) saveToBatch(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	funcName string,
	fn *Method,
	sender *proto.Address,
	args []string,
	nonce uint64,
) error {
	log := logger.Logger()
	txID := stub.GetTxID()

	argsToValidate := args
	if fn.needsAuth {
		argsToValidate = append([]string{sender.AddrString()}, args...)
	}

	if err := reflectx.ValidateArguments(cc.contract, fn.Name, stub, argsToValidate...); err != nil {
		return err
	}

	key, err := stub.CreateCompositeKey(config.BatchPrefix, []string{txID})
	if err != nil {
		log.Errorf("Couldn't create composite key for tx %s: %s", txID, err.Error())
		return err
	}

	txTimestamp, err := stub.GetTxTimestamp()
	if err != nil {
		log.Errorf("Couldn't get timestamp for tx %s: %s", txID, err.Error())
		return err
	}

	pending := &proto.PendingTx{
		Method:    funcName,
		Sender:    sender,
		Args:      args,
		Timestamp: txTimestamp.GetSeconds(),
		Nonce:     nonce,
	}

	carrier := cc.contract.TracingHandler().RemoteCarrier(traceCtx)

	// Sorting carrier keys in alphabetical order
	// and packing carrier data into key-value pairs of preimage transaction
	keys := carrier.Keys()
	if len(keys) != 0 {
		sort.Strings(keys)
		var pairs []*proto.Pair
		for _, k := range keys {
			pairs = append(pairs, &proto.Pair{
				Key:   k,
				Value: carrier.Get(k),
			})
		}
		pending.Pairs = pairs
	}

	data, err := pb.Marshal(pending)
	if err != nil {
		log.Errorf("Couldn't marshal transaction %s: %s", txID, err.Error())
		return err
	}

	return stub.PutState(key, data)
}

func (cc *ChainCode) loadFromBatch(
	stub shim.ChaincodeStubInterface,
	txID string,
) (*proto.PendingTx, string, error) {
	log := logger.Logger()
	key, err := stub.CreateCompositeKey(config.BatchPrefix, []string{txID})
	if err != nil {
		log.Errorf("Couldn't create composite key for tx %s: %s", txID, err.Error())
		return nil, "", err
	}
	data, err := stub.GetState(key)
	if err != nil {
		log.Errorf("Couldn't load transaction %s from state: %s", txID, err.Error())
		return nil, "", err
	}
	if len(data) == 0 {
		log.Warningf("Transaction %s not found", txID)
		return nil, "", fmt.Errorf("transaction %s not found", txID)
	}

	defer func() {
		err = stub.DelState(key)
		if err != nil {
			log.Errorf("Couldn't delete from state tx %s: %s", txID, err.Error())
		}
	}()

	pending := new(proto.PendingTx)
	if err = pb.Unmarshal(data, pending); err != nil {
		log.Errorf("couldn't unmarshal transaction %s: %s", txID, err.Error())
		return nil, key, err
	}

	method, err := cc.Method(pending.GetMethod())
	if err != nil {
		log.Errorf("unknown method %s in tx %s", pending.GetMethod(), txID)
		return pending, key, fmt.Errorf("unknown method %s in tx %s", pending.GetMethod(), txID)
	}

	if !method.needsAuth {
		return pending, key, nil
	}

	if pending.GetSender() == nil {
		log.Errorf("no sender in tx %s", txID)
		return pending, key, fmt.Errorf("no sender in tx %s", txID)
	}

	sender := types.NewSenderFromAddr((*types.Address)(pending.GetSender()))
	if err = checkNonce(stub, sender, pending.GetNonce()); err != nil {
		log.Errorf("incorrect tx %s nonce: %s", txID, err.Error())
		return pending, key, err
	}

	return pending, key, nil
}

//nolint:funlen
func (cc *ChainCode) batchExecute(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	dataIn string,
	cfgBytes []byte,
) peer.Response {
	traceCtx, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, BatchExecute)
	defer span.End()

	log := logger.Logger()
	batchID := stub.GetTxID()
	span.SetAttributes(attribute.String("batch_tx_id", batchID))
	btchStub := cachestub.NewBatchCacheStub(stub)
	start := time.Now()
	defer func() {
		log.Infof("batch %s elapsed time %d ms", batchID, time.Since(start).Milliseconds())
	}()
	response := proto.BatchResponse{}
	events := proto.BatchEvent{}
	var batch proto.Batch
	if err := pb.Unmarshal([]byte(dataIn), &batch); err != nil {
		log.Errorf("Couldn't unmarshal batch %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}

	span.AddEvent("handle transactions in batch")
	ids := make([]string, 0, len(batch.GetTxIDs()))
	for _, txID := range batch.GetTxIDs() {
		ids = append(ids, hex.EncodeToString(txID))
		resp, event := cc.batchedTxExecute(traceCtx, btchStub, txID, cfgBytes)
		response.TxResponses = append(response.TxResponses, resp)
		events.Events = append(events.Events, event)
	}
	span.SetAttributes(attribute.StringSlice("ids", ids))

	if !cc.contract.ContractConfig().GetOptions().GetDisableSwaps() {
		span.AddEvent("handle swaps")
		for _, s := range batch.GetSwaps() {
			response.SwapResponses = append(response.SwapResponses, swap.Answer(btchStub, s, robotSideTimeout))
		}
		for _, swapKey := range batch.GetKeys() {
			response.SwapKeyResponses = append(response.SwapKeyResponses, swap.RobotDone(btchStub, swapKey.GetId(), swapKey.GetKey()))
		}
	}

	if !cc.contract.ContractConfig().GetOptions().GetDisableMultiSwaps() {
		span.AddEvent("handle multi-swaps")
		for _, s := range batch.GetMultiSwaps() {
			response.SwapResponses = append(response.SwapResponses, multiswap.Answer(btchStub, s, robotSideTimeout))
		}
		for _, swapKey := range batch.GetMultiSwapsKeys() {
			response.SwapKeyResponses = append(response.SwapKeyResponses, multiswap.RobotDone(btchStub, swapKey.GetId(), swapKey.GetKey()))
		}
	}

	span.AddEvent("commit")
	if err := btchStub.Commit(); err != nil {
		log.Errorf("Couldn't commit batch %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}

	response.CreatedSwaps = btchStub.Swaps
	response.CreatedMultiSwap = btchStub.MultiSwaps

	data, err := pb.Marshal(&response)
	if err != nil {
		log.Errorf("Couldn't marshal batch response %s: %s", batchID, err.Error())
		span.SetStatus(codes.Error, "marshalling batch response failed")

		return shim.Error(err.Error())
	}
	eventData, err := pb.Marshal(&events)
	if err != nil {
		log.Errorf("Couldn't marshal batch event %s: %s", batchID, err.Error())
		span.SetStatus(codes.Error, "marshalling batch event failed")

		return shim.Error(err.Error())
	}
	if err = stub.SetEvent(BatchExecute, eventData); err != nil {
		log.Errorf("Couldn't set batch event %s: %s", batchID, err.Error())
		span.SetStatus(codes.Error, "set batch event failed")

		return shim.Error(err.Error())
	}

	span.SetStatus(codes.Ok, "")

	return shim.Success(data)
}

type TxResponse struct {
	Method     string                    `json:"method"`
	Error      string                    `json:"error,omitempty"`
	Result     string                    `json:"result"`
	Events     map[string][]byte         `json:"events,omitempty"`
	Accounting []*proto.AccountingRecord `json:"accounting"`
}

func (cc *ChainCode) batchedTxExecute(
	traceCtx telemetry.TraceContext,
	stub *cachestub.BatchCacheStub,
	binaryTxID []byte,
	cfgBytes []byte,
) (r *proto.TxResponse, e *proto.BatchTxEvent) {
	traceCtx, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "batchTxExecute")
	defer span.End()

	log := logger.Logger()
	start := time.Now()
	methodName := "unknown"
	span.SetAttributes(attribute.String("method", methodName))

	txID := hex.EncodeToString(binaryTxID)
	span.SetAttributes(attribute.String("preimage_tx_id", txID))
	defer func() {
		log.Infof("batched method %s txid %s elapsed time %d ms", methodName, txID, time.Since(start).Milliseconds())
	}()

	r = &proto.TxResponse{Id: binaryTxID, Error: &proto.ResponseError{Error: "panic batchedTxExecute"}}
	e = &proto.BatchTxEvent{Id: binaryTxID, Error: &proto.ResponseError{Error: "panic batchedTxExecute"}}
	defer func() {
		if rc := recover(); rc != nil {
			log.Criticalf("Tx %s panicked:\n%s", txID, string(debug.Stack()))
		}
	}()

	span.AddEvent("load from batch")
	pending, key, err := cc.loadFromBatch(stub, txID)
	if err != nil && pending != nil {
		if delErr := stub.ChaincodeStubInterface.DelState(key); delErr != nil {
			log.Errorf("failed deleting key %s from state on txId: %s", key, delErr.Error())
		}
		ee := proto.ResponseError{Error: "function and args loading error: " + err.Error()}
		span.SetStatus(codes.Error, err.Error())
		return &proto.TxResponse{
				Id:     binaryTxID,
				Method: pending.GetMethod(),
				Error:  &ee,
			}, &proto.BatchTxEvent{
				Id:     binaryTxID,
				Method: pending.GetMethod(),
				Error:  &ee,
			}
	} else if err != nil {
		if delErr := stub.ChaincodeStubInterface.DelState(key); delErr != nil {
			log.Errorf("failed deleting key %s from state: %s", key, delErr.Error())
		}
		ee := proto.ResponseError{Error: "function and args loading error: " + err.Error()}
		span.SetStatus(codes.Error, err.Error())
		return &proto.TxResponse{
				Id:    binaryTxID,
				Error: &ee,
			}, &proto.BatchTxEvent{
				Id:    binaryTxID,
				Error: &ee,
			}
	}

	txStub := stub.NewTxCacheStub(txID)
	method, err := cc.Method(pending.GetMethod())
	if err != nil {
		msg := fmt.Sprintf("parsing method '%s' in tx '%s': %s", pending.GetMethod(), txID, err.Error())
		span.SetStatus(codes.Error, msg)
		log.Info(msg)

		_ = stub.ChaincodeStubInterface.DelState(key)
		ee := proto.ResponseError{Error: "unknown method " + pending.GetMethod()}
		return &proto.TxResponse{
				Id:     binaryTxID,
				Method: pending.GetMethod(),
				Error:  &ee,
			}, &proto.BatchTxEvent{
				Id:     binaryTxID,
				Method: pending.GetMethod(),
				Error:  &ee,
			}
	}
	methodName = pending.GetMethod()
	span.SetAttributes(attribute.String("method", methodName))

	if len(pending.GetPairs()) != 0 {
		carrier := propagation.MapCarrier{}
		for _, pair := range pending.GetPairs() {
			carrier.Set(pair.GetKey(), pair.GetValue())
		}

		traceCtx = cc.contract.TracingHandler().ExtractContext(carrier)
	}

	span.AddEvent("calling method")
	response, err := cc.callMethod(traceCtx, txStub, method, pending.GetSender(), pending.GetArgs(), cfgBytes)
	if err != nil {
		_ = stub.ChaincodeStubInterface.DelState(key)
		ee := proto.ResponseError{Error: err.Error()}
		span.SetStatus(codes.Error, "call method returned error")

		return &proto.TxResponse{
				Id:     binaryTxID,
				Method: pending.GetMethod(),
				Error:  &ee,
			}, &proto.BatchTxEvent{
				Id:     binaryTxID,
				Method: pending.GetMethod(),
				Error:  &ee,
			}
	}

	span.AddEvent("commit")
	writes, events := txStub.Commit()

	sort.Slice(txStub.Accounting, func(i, j int) bool {
		return strings.Compare(txStub.Accounting[i].String(), txStub.Accounting[j].String()) < 0
	})

	span.SetStatus(codes.Ok, "")

	return &proto.TxResponse{
			Id:     binaryTxID,
			Method: pending.GetMethod(),
			Writes: writes,
		},
		&proto.BatchTxEvent{
			Id:         binaryTxID,
			Method:     pending.GetMethod(),
			Accounting: txStub.Accounting,
			Events:     events,
			Result:     response,
		}
}
