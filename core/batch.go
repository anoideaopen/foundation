package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

const batchKey = "batchTransactions"

func (cc *ChainCode) saveToBatch(
	stub shim.ChaincodeStubInterface,
	fn string,
	sender *proto.Address,
	args []string,
	nonce uint64,
) error {
	logger := Logger()
	txID := stub.GetTxID()
	method, exists := cc.methods[fn]
	if !exists {
		return fmt.Errorf("method '%s' not found", fn)
	}
	_, err := doConvertToCall(stub, method, args)
	if err != nil {
		return fmt.Errorf("validate arguments. %w", err)
	}
	key, err := stub.CreateCompositeKey(cc.batchPrefix, []string{txID})
	if err != nil {
		logger.Errorf("Couldn't create composite key for tx %s: %s", txID, err.Error())
		return err
	}

	txTimestamp, err := stub.GetTxTimestamp()
	if err != nil {
		logger.Errorf("Couldn't get timestamp for tx %s: %s", txID, err.Error())
		return err
	}

	data, err := pb.Marshal(&proto.PendingTx{
		Method:    fn,
		Sender:    sender,
		Args:      args,
		Timestamp: txTimestamp.Seconds,
		Nonce:     nonce,
	})
	if err != nil {
		logger.Errorf("Couldn't marshal transaction %s: %s", txID, err.Error())
		return err
	}
	return stub.PutState(key, data)
}

func (cc *ChainCode) loadFromBatch( //nolint:funlen
	stub shim.ChaincodeStubInterface,
	txID string,
	batchTimestamp int64,
) (*proto.PendingTx, string, error) {
	logger := Logger()
	key, err := stub.CreateCompositeKey(cc.batchPrefix, []string{txID})
	if err != nil {
		logger.Errorf("Couldn't create composite key for tx %s: %s", txID, err.Error())
		return nil, "", err
	}
	data, err := stub.GetState(key)
	if err != nil {
		logger.Errorf("Couldn't load transaction %s from state: %s", txID, err.Error())
		return nil, "", err
	}
	if len(data) == 0 {
		logger.Warningf("Transaction %s not found", txID)
		return nil, "", fmt.Errorf("transaction %s not found", txID)
	}

	defer func() {
		err = stub.DelState(key)
		if err != nil {
			logger.Errorf("Couldn't delete from state tx %s: %s", txID, err.Error())
		}
	}()

	pending := new(proto.PendingTx)
	if err = pb.Unmarshal(data, pending); err != nil {
		var args []string
		if err = json.Unmarshal(data, &args); err != nil {
			logger.Errorf("Couldn't unmarshal transaction %s: %s", txID, err.Error())
			return nil, key, err
		}

		pending = &proto.PendingTx{
			Method: args[0],
			Args:   args[2:],
		}
	}

	if cc.txTTL > 0 && batchTimestamp-pending.Timestamp > int64(cc.txTTL) {
		logger.Errorf("Transaction ttl expired %s", txID)
		return pending, key, fmt.Errorf("transaction expired. Transaction %s batchTimestamp-pending.Timestamp %d more than %d",
			txID, batchTimestamp-pending.Timestamp, cc.txTTL)
	}

	if cc.nonceTTL != 0 {
		method, exists := cc.methods[pending.Method]
		if !exists {
			logger.Errorf("unknown method %s in tx %s", pending.Method, txID)
			return pending, key, fmt.Errorf("unknown method %s in tx %s", pending.Method, txID)
		}

		if !method.needsAuth {
			return pending, key, nil
		}

		if pending.Sender == nil {
			logger.Errorf("no sender in tx %s", txID)
			return pending, key, fmt.Errorf("no sender in tx %s", txID)
		}
		if err = cc.nonceCheckFn(stub, types.NewSenderFromAddr((*types.Address)(pending.Sender)), pending.Nonce); err != nil {
			logger.Errorf("incorrect tx %s nonce: %s", txID, err.Error())
			return pending, key, err
		}
	}

	return pending, key, nil
}

//nolint:funlen
func (cc *ChainCode) batchExecute(
	stub shim.ChaincodeStubInterface,
	dataIn string,
	platformSKI []byte,
	initArgs []string,
) peer.Response {
	logger := Logger()
	batchID := stub.GetTxID()
	btchStub := newBatchStub(stub)
	start := time.Now()
	defer func() {
		logger.Infof("batch %s elapsed time %d ms", batchID, time.Since(start).Milliseconds())
	}()
	response := proto.BatchResponse{}
	events := proto.BatchEvent{}
	var batch proto.Batch
	if err := pb.Unmarshal([]byte(dataIn), &batch); err != nil {
		logger.Errorf("Couldn't unmarshal batch %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}

	batchTimestamp, err := stub.GetTxTimestamp()
	if err != nil {
		logger.Errorf("Couldn't get batch timestamp %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}

	for _, txID := range batch.TxIDs {
		resp, event := cc.batchedTxExecute(btchStub, txID, batchTimestamp.Seconds, platformSKI, initArgs)
		response.TxResponses = append(response.TxResponses, resp)
		events.Events = append(events.Events, event)
	}

	if !cc.disableSwaps {
		for _, swap := range batch.Swaps {
			response.SwapResponses = append(response.SwapResponses, swapAnswer(btchStub, swap))
		}
		for _, swapKey := range batch.Keys {
			response.SwapKeyResponses = append(response.SwapKeyResponses, swapRobotDone(btchStub, swapKey.Id, swapKey.Key))
		}
	}

	if !cc.disableMultiSwaps {
		for _, swap := range batch.MultiSwaps {
			response.SwapResponses = append(response.SwapResponses, multiSwapAnswer(btchStub, swap))
		}
		for _, swapKey := range batch.MultiSwapsKeys {
			response.SwapKeyResponses = append(response.SwapKeyResponses, multiSwapRobotDone(btchStub, swapKey.Id, swapKey.Key))
		}
	}

	if err = btchStub.Commit(); err != nil {
		logger.Errorf("Couldn't commit batch %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}

	response.CreatedSwaps = btchStub.swaps
	response.CreatedMultiSwap = btchStub.multiSwaps

	data, err := pb.Marshal(&response)
	if err != nil {
		logger.Errorf("Couldn't marshal batch response %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}
	eventData, err := pb.Marshal(&events)
	if err != nil {
		logger.Errorf("Couldn't marshal batch event %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}
	if err = stub.SetEvent("batchExecute", eventData); err != nil {
		logger.Errorf("Couldn't set batch event %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}
	return shim.Success(data)
}

// TxResponse is a response for a single transaction in a batch
type TxResponse struct {
	Method     string                    `json:"method"`
	Error      string                    `json:"error,omitempty"`
	Result     string                    `json:"result"`
	Events     map[string][]byte         `json:"events,omitempty"`
	Accounting []*proto.AccountingRecord `json:"accounting"`
}

func (cc *ChainCode) batchedTxExecute( //nolint:funlen
	stub *batchStub,
	binaryTxID []byte,
	batchTimestamp int64,
	platformSKI []byte,
	initArgs []string,
) (r *proto.TxResponse, e *proto.BatchTxEvent) {
	logger := Logger()
	start := time.Now()
	methodName := "unknown"

	txID := hex.EncodeToString(binaryTxID)
	defer func() {
		logger.Infof("batched method %s txid %s elapsed time %d ms", methodName, txID, time.Since(start).Milliseconds())
	}()

	r = &proto.TxResponse{Id: binaryTxID, Error: &proto.ResponseError{Error: "panic batchedTxExecute"}}
	e = &proto.BatchTxEvent{Id: binaryTxID, Error: &proto.ResponseError{Error: "panic batchedTxExecute"}}
	defer func() {
		if rc := recover(); rc != nil {
			logger.Criticalf("Tx %s panicked:\n%s", txID, string(debug.Stack()))
		}
	}()

	pending, key, err := cc.loadFromBatch(stub, txID, batchTimestamp)
	if err != nil && pending != nil {
		_ = stub.ChaincodeStubInterface.DelState(key)
		ee := proto.ResponseError{Error: fmt.Sprintf("function and args loading error: %s", err.Error())}
		return &proto.TxResponse{Id: binaryTxID, Method: pending.Method, Error: &ee}, &proto.BatchTxEvent{Id: binaryTxID, Method: pending.Method, Error: &ee}
	} else if err != nil {
		_ = stub.ChaincodeStubInterface.DelState(key)
		ee := proto.ResponseError{Error: fmt.Sprintf("function and args loading error: %s", err.Error())}
		return &proto.TxResponse{Id: binaryTxID, Error: &ee}, &proto.BatchTxEvent{Id: binaryTxID, Error: &ee}
	}

	txStub := stub.newTxStub(txID)
	method, exists := cc.methods[pending.Method]
	if !exists {
		logger.Infof("Unknown method %s in tx %s", pending.Method, txID)
		_ = stub.ChaincodeStubInterface.DelState(key)
		ee := proto.ResponseError{Error: fmt.Sprintf("unknown method %s", pending.Method)}
		return &proto.TxResponse{Id: binaryTxID, Method: pending.Method, Error: &ee}, &proto.BatchTxEvent{Id: binaryTxID, Method: pending.Method, Error: &ee}
	}
	methodName = pending.Method

	response, err := cc.callMethod(txStub, method, pending.Sender, pending.Args, platformSKI, initArgs)
	if err != nil {
		_ = stub.ChaincodeStubInterface.DelState(key)
		ee := proto.ResponseError{Error: err.Error()}
		return &proto.TxResponse{Id: binaryTxID, Method: pending.Method, Error: &ee}, &proto.BatchTxEvent{Id: binaryTxID, Method: pending.Method, Error: &ee}
	}

	writes, events := txStub.Commit()

	sort.Slice(txStub.accounting, func(i, j int) bool {
		return strings.Compare(txStub.accounting[i].String(), txStub.accounting[j].String()) < 0
	})

	return &proto.TxResponse{
			Id:     binaryTxID,
			Method: pending.Method,
			Writes: writes,
		},
		&proto.BatchTxEvent{
			Id:         binaryTxID,
			Method:     pending.Method,
			Accounting: txStub.accounting,
			Events:     events,
			Result:     response,
		}
}
