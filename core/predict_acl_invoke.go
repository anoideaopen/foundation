package core

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"

	"github.com/anoideaopen/foundation/core/contract"
	"github.com/anoideaopen/foundation/core/helpers"
	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

const (
	FnCheckKeys      = "checkKeys"
	FnGetAccountInfo = "getAccountInfo"
	FnCheckAddress   = "checkAddress"

	FnTransfer = "transfer"
)

func predictACLCalls(stub shim.ChaincodeStubInterface, tasks []*proto.Task, chaincode *Chaincode) {
	callsMap := make(map[string]map[string]struct{})
	methods := chaincode.Router().Methods()
	wg := &sync.WaitGroup{}
	for _, task := range tasks {
		if task == nil {
			break
		}
		wg.Add(1)
		go func(task *proto.Task) {
			defer wg.Done()
			method := methods[task.GetMethod()]

			predictTaskACLCalls(chaincode, task, method, callsMap)
		}(task)
	}
	wg.Wait()

	var requestBytes [][]byte
	for fn, mArg := range callsMap {
		for arg := range mArg {
			logger.Logger().Debug("PredictAclCalls txID %s, fn: '%s', arg '%s'\n", stub.GetTxID(), fn, arg)
			bytes, err := json.Marshal([]string{fn, arg})
			if err == nil {
				requestBytes = append(requestBytes, bytes)
			} else {
				logger.Logger().Errorf("PredictAclCalls txID %s, failed to marshal, fn: '%s', arg '%s': %v",
					stub.GetTxID(), fn, arg, err)
			}
		}
	}

	_, err := helpers.GetAccountsInfo(stub, requestBytes)
	if err != nil {
		logger.Logger().Errorf("PredictAclCalls txID %s, failed to invoke acl calls: %v", stub.GetTxID(), err)
	}
}

func predictTaskACLCalls(chaincode *Chaincode, task *proto.Task, method contract.Method, callsMap map[string]map[string]struct{}) {
	signers := getSigners(method, task)
	if signers != nil {
		addACLCall(callsMap, FnCheckKeys, strings.Join(signers, "/"))
	}

	inputVal := reflect.ValueOf(chaincode.contract)
	methodVal := inputVal.MethodByName(method.MethodName)
	if !methodVal.IsValid() {
		return
	}

	methodArgs := task.GetArgs()[3 : 3+(method.NumArgs-1)]
	methodType := methodVal.Type()

	if methodType.NumIn()-len(signers) != len(methodArgs) {
		return
	}
	if strings.Contains(task.GetMethod(), FnTransfer) && len(methodArgs) > 0 {
		addACLCall(callsMap, FnCheckAddress, methodArgs[0])
	}

	for i, arg := range methodArgs {
		t := methodType.In(i)
		if t.Kind() != reflect.Pointer {
			continue
		}
		argInterface := reflect.New(t.Elem()).Interface()
		_, ok := argInterface.(*types.Address)
		if ok {
			continue
		}
		_, err := types.AddrFromBase58Check(arg)
		if err != nil {
			continue
		}
		addACLCall(callsMap, FnGetAccountInfo, arg)
	}
}

func addACLCall(callsMap map[string]map[string]struct{}, method string, arg string) {
	logger.Logger().Info("add acl call method %s arg %s", method, arg)
	_, ok := callsMap[method]
	if !ok {
		callsMap[method] = map[string]struct{}{}
	}
	callsMap[method][arg] = struct{}{}
}

func getSigners(method contract.Method, task *proto.Task) []string {
	if !method.RequiresAuth {
		return nil
	}

	invocation, err := parseInvocationDetails(method, task.GetArgs())
	if err != nil {
		return nil
	}

	return invocation.signatureArgs[:invocation.signersCount]
}
