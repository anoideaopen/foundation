package core

import (
	"encoding/hex"
	"errors"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/config"
	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/core/multiswap"
	"github.com/anoideaopen/foundation/core/swap"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	pb "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const robotSideTimeout = 300 // 5 minutes

func (cc *Chaincode) saveToBatch(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	function string,
	sender *proto.Address,
	args []string,
	nonce uint64,
) error {
	log := logger.Logger()
	txID := stub.GetTxID()

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
		Method:    function,
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

func (cc *Chaincode) getBatchFromState(stub shim.ChaincodeStubInterface, batch *proto.Batch) error {
	log := logger.Logger()

	keys, err := cc.collectKeysOfBatch(stub, batch)
	if err != nil {
		log.Errorf("couldn't collect kyes of batch: %s", err.Error())
		return err
	}

	result, err := stub.GetMultipleStates(keys...)
	if err != nil {
		log.Errorf("couldn't get multiple states: %s", err.Error())
		return err
	}

	if len(result) != len(keys) {
		return errors.New("len of result is not equal to len of keys")
	}

	err = cc.parseResponseFromBatchKeys(batch, result)
	if err != nil {
		log.Errorf("couldn't parsing response from batch keys: %s", err.Error())
		return err
	}

	return nil
}

func (cc *Chaincode) collectKeysOfBatch(stub shim.ChaincodeStubInterface, batch *proto.Batch) ([]string, error) {
	log := logger.Logger()

	keys := make([]string, 0, len(batch.GetTxIDs())+len(batch.GetKeys())+len(batch.GetMultiSwapsKeys()))

	for _, b := range batch.GetTxIDs() {
		txID := hex.EncodeToString(b)
		key, err := stub.CreateCompositeKey(config.BatchPrefix, []string{txID})
		if err != nil {
			log.Errorf("Couldn't create composite key for tx %s: %s", txID, err.Error())
			return nil, err
		}
		keys = append(keys, key)
	}

	for _, sk := range batch.GetKeys() {
		txID := hex.EncodeToString(sk.GetId())
		key, err := stub.CreateCompositeKey(swap.SwapCompositeType, []string{txID})
		if err != nil {
			log.Errorf("Couldn't create composite key for tx %s: %s", txID, err.Error())
			return nil, err
		}
		keys = append(keys, key)
	}

	for _, msk := range batch.GetMultiSwapsKeys() {
		txID := hex.EncodeToString(msk.GetId())
		key, err := stub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{txID})
		if err != nil {
			log.Errorf("couldn't create composite key for tx %s: %s", txID, err.Error())
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func (cc *Chaincode) parseResponseFromBatchKeys(batch *proto.Batch, result [][]byte) error {
	log := logger.Logger()

	batch.Pendings = make([]*proto.PendingTx, 0, len(batch.GetTxIDs()))
	for i, b := range batch.GetTxIDs() {
		if result[i] == nil {
			batch.Pendings = append(batch.Pendings, nil)
			continue
		}
		txID := hex.EncodeToString(b)
		pending := new(proto.PendingTx)
		if err := pb.Unmarshal(result[i], pending); err != nil {
			log.Errorf("couldn't unmarshal transaction %s: %s", txID, err.Error())
			return err
		}
		batch.Pendings = append(batch.Pendings, pending)
	}

	result = result[len(batch.GetTxIDs()):]

	for i, sk := range batch.GetKeys() {
		sks := &proto.SwapKey_Swap{}
		batch.GetKeys()[i].Payload = sks

		if result[i] == nil {
			continue
		}

		txID := hex.EncodeToString(sk.GetId())
		s := new(proto.Swap)
		if err := pb.Unmarshal(result[i], s); err != nil {
			log.Errorf("couldn't unmarshal transaction %s: %s", txID, err.Error())
			return err
		}

		sks.Swap = s
	}

	result = result[len(batch.GetKeys()):]

	for i, sk := range batch.GetMultiSwapsKeys() {
		skms := &proto.SwapKey_MultiSwap{}
		batch.GetMultiSwapsKeys()[i].Payload = skms

		if result[i] == nil {
			continue
		}

		txID := hex.EncodeToString(sk.GetId())
		ms := new(proto.MultiSwap)
		if err := pb.Unmarshal(result[i], ms); err != nil {
			log.Errorf("couldn't unmarshal transaction %s: %s", txID, err.Error())
			return err
		}

		skms.MultiSwap = ms
	}

	return nil
}

func (cc *Chaincode) checkPending(
	stub shim.ChaincodeStubInterface,
	txID string,
	pending *proto.PendingTx,
) error {
	log := logger.Logger()

	if pending == nil {
		log.Warningf("transaction %s not found", txID)
		return fmt.Errorf("transaction %s not found", txID)
	}

	key, err := stub.CreateCompositeKey(config.BatchPrefix, []string{txID})
	if err != nil {
		log.Errorf("Couldn't create composite key for tx %s: %s", txID, err.Error())
		return err
	}

	err = stub.DelState(key)
	if err != nil {
		log.Errorf("couldn't delete from state tx %s: %s", txID, err.Error())
	}

	method := cc.Router().Method(pending.GetMethod())
	if method == "" {
		log.Errorf("unknown method %s in tx %s", pending.GetMethod(), txID)
		return fmt.Errorf("unknown method %s in tx %s", pending.GetMethod(), txID)
	}

	if !cc.Router().AuthRequired(method) {
		return nil
	}

	if pending.GetSender() == nil {
		log.Errorf("no sender in tx %s", txID)
		return fmt.Errorf("no sender in tx %s", txID)
	}

	sender := types.NewSenderFromAddr((*types.Address)(pending.GetSender()))
	n := new(Nonce)
	if err = n.check(stub, sender, pending.GetNonce()); err != nil {
		log.Errorf("incorrect tx %s nonce: %s", txID, err.Error())
		return err
	}

	return nil
}

//nolint:funlen
func (cc *Chaincode) batchExecute(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	dataIn string,
) *peer.Response {
	traceCtx, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, BatchExecute)
	defer span.End()

	log := logger.Logger()
	batchID := stub.GetTxID()
	span.SetAttributes(attribute.String("batch_tx_id", batchID))
	batchStub := cachestub.NewBatchCacheStub(stub)
	start := time.Now()
	defer func() {
		log.Infof("batch: tx id: %s, elapsed: %s", batchID, time.Since(start))
	}()
	response := proto.BatchResponse{}
	events := proto.BatchEvent{}
	var batch proto.Batch
	if err := pb.Unmarshal([]byte(dataIn), &batch); err != nil {
		log.Errorf("Couldn't unmarshal batch %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}

	batchTxTime, err := batchStub.GetTxTimestamp()
	if err != nil {
		log.Errorf("couldn't get timestamp for batch %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}

	log.Warningf("batch: tx id: %s, txs: %d", batchID, len(batch.GetTxIDs()))

	if err = cc.getBatchFromState(stub, &batch); err != nil {
		log.Errorf("couldn't get batch %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}

	span.AddEvent("handle transactions in batch")
	ids := make([]string, 0, len(batch.GetTxIDs()))
	for txIdx, txID := range batch.GetTxIDs() {
		txTimestamp := &timestamppb.Timestamp{
			Seconds: batchTxTime.GetSeconds(),
			Nanos:   batchTxTime.GetNanos() + int32(txIdx),
		}
		ids = append(ids, hex.EncodeToString(txID))
		resp, event := cc.batchedTxExecute(traceCtx, batchStub, txID, txTimestamp, batch.GetPendings()[txIdx])
		response.TxResponses = append(response.TxResponses, resp)
		events.Events = append(events.Events, event)
	}
	span.SetAttributes(attribute.StringSlice("ids", ids))

	if !cc.contract.ContractConfig().GetOptions().GetDisableSwaps() {
		span.AddEvent("handle swaps")
		for _, s := range batch.GetSwaps() {
			response.SwapResponses = append(response.SwapResponses, swap.Answer(batchStub, s, robotSideTimeout))
		}
		for _, swapKey := range batch.GetKeys() {
			s, _ := swapKey.GetPayload().(*proto.SwapKey_Swap)
			response.SwapKeyResponses = append(response.SwapKeyResponses, swap.RobotDone(batchStub, swapKey.GetId(), swapKey.GetKey(), s.Swap))
		}
	}

	if !cc.contract.ContractConfig().GetOptions().GetDisableMultiSwaps() {
		span.AddEvent("handle multi-swaps")
		for _, s := range batch.GetMultiSwaps() {
			response.SwapResponses = append(response.SwapResponses, multiswap.Answer(batchStub, s, robotSideTimeout))
		}
		for _, swapKey := range batch.GetMultiSwapsKeys() {
			ms, _ := swapKey.GetPayload().(*proto.SwapKey_MultiSwap)
			response.SwapKeyResponses = append(response.SwapKeyResponses, multiswap.RobotDone(batchStub, swapKey.GetId(), swapKey.GetKey(), ms.MultiSwap))
		}
	}

	span.AddEvent("commit")
	if err := batchStub.Commit(); err != nil {
		log.Errorf("Couldn't commit batch %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}

	response.CreatedSwaps = batchStub.Swaps
	response.CreatedMultiSwap = batchStub.MultiSwaps

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

func (cc *Chaincode) batchedTxExecute(
	traceCtx telemetry.TraceContext,
	stub *cachestub.BatchCacheStub,
	binaryTxID []byte,
	txTimestamp *timestamppb.Timestamp,
	pending *proto.PendingTx,
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
	err := cc.checkPending(stub, txID, pending)
	if err != nil {
		ee := proto.ResponseError{Error: "function and args loading error: " + err.Error()}
		span.SetStatus(codes.Error, err.Error())
		return &proto.TxResponse{Id: binaryTxID, Method: pending.GetMethod(), Error: &ee},
			&proto.BatchTxEvent{Id: binaryTxID, Method: pending.GetMethod(), Error: &ee}
	}

	txStub := stub.NewTxCacheStub(txID, txTimestamp)
	method := cc.Router().Method(pending.GetMethod())
	if method == "" {
		msg := fmt.Sprintf(
			"parsing method '%s' in tx '%s': method '%s' not found",
			pending.GetMethod(),
			txID,
			pending.GetMethod(),
		)
		span.SetStatus(codes.Error, msg)
		log.Info(msg)

		ee := proto.ResponseError{Error: "unknown method " + pending.GetMethod()}
		return &proto.TxResponse{Id: binaryTxID, Method: pending.GetMethod(), Error: &ee},
			&proto.BatchTxEvent{Id: binaryTxID, Method: pending.GetMethod(), Error: &ee}
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
	response, err := cc.InvokeContractMethod(traceCtx, txStub, pending.GetSender(), method, pending.GetArgs())
	if err != nil {
		ee := proto.ResponseError{Error: err.Error()}
		span.SetStatus(codes.Error, "call method returned error")

		return &proto.TxResponse{Id: binaryTxID, Method: pending.GetMethod(), Error: &ee},
			&proto.BatchTxEvent{Id: binaryTxID, Method: pending.GetMethod(), Error: &ee}
	}

	span.AddEvent("commit")
	writes, events := txStub.Commit()

	sort.Slice(txStub.Accounting, func(i, j int) bool {
		return strings.Compare(txStub.Accounting[i].String(), txStub.Accounting[j].String()) < 0
	})

	span.SetStatus(codes.Ok, "")

	return &proto.TxResponse{Id: binaryTxID, Method: pending.GetMethod(), Writes: writes},
		&proto.BatchTxEvent{
			Id: binaryTxID, Method: pending.GetMethod(),
			Accounting: txStub.Accounting, Events: events, Result: response,
		}
}
