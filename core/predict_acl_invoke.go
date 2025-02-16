package core

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"

	"github.com/anoideaopen/foundation/core/helpers"
	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
)

const (
	FnTransfer = "transfer"
)

type predictACL struct {
	stub             shim.ChaincodeStubInterface
	invocationBuffer map[string][]byte // [method + arg] -> invocation bytes
	mu               sync.RWMutex
}

func predictPendingsACLCalls(stub shim.ChaincodeStubInterface, pendings []*proto.PendingTx, chaincode *Chaincode) {
	p := predictACL{
		stub:             stub,
		invocationBuffer: make(map[string][]byte),
		mu:               sync.RWMutex{},
	}

	wg := &sync.WaitGroup{}

	for _, pending := range pendings {
		if pending == nil {
			continue
		}

		wg.Add(1)
		go func(pending *proto.PendingTx) {
			defer wg.Done()

			p.predictPendingACLCalls(chaincode, pending)
		}(pending)
	}

	wg.Wait()

	if len(p.invocationBuffer) == 0 {
		return
	}

	requestBytes := make([][]byte, len(p.invocationBuffer))
	i := 0
	for _, bytes := range p.invocationBuffer {
		requestBytes[i] = bytes
		i++
	}

	_, err := helpers.GetAccountsInfo(stub, requestBytes)
	if err != nil {
		logger.Logger().Errorf("PredictAclCalls txID %s, failed to invoke acl calls: %v", stub.GetTxID(), err)
	}
}

func (p *predictACL) predictPendingACLCalls(chaincode *Chaincode, pending *proto.PendingTx) {
	var (
		method       = chaincode.Router().Method(pending.GetMethod())
		authRequired = chaincode.Router().AuthRequired(method)
		args         = pending.GetArgs()
	)

	if !strings.Contains(pending.GetMethod(), FnTransfer) || !authRequired {
		return
	}

	inputVal := reflect.ValueOf(chaincode.contract)
	methodVal := inputVal.MethodByName(method)
	if !methodVal.IsValid() {
		return
	}

	methodType := methodVal.Type()

	// check method input args without signer, to skip signers in future for
	if methodType.NumIn()-1 != len(args) {
		return
	}
	if strings.Contains(pending.GetMethod(), FnTransfer) && len(args) > 0 {
		t := methodType.In(1)
		if t.Kind() != reflect.Pointer {
			return
		}

		argInterface := reflect.New(t.Elem()).Interface()
		_, ok := argInterface.(*types.Address)
		if !ok {
			return
		}

		p.addCall(helpers.FnCheckAddress, args[0])
	}
}

func predictTasksACLCalls(stub shim.ChaincodeStubInterface, tasks []*proto.Task, chaincode *Chaincode) {
	p := predictACL{
		stub:             stub,
		invocationBuffer: make(map[string][]byte),
		mu:               sync.RWMutex{},
	}

	wg := &sync.WaitGroup{}

	for _, task := range tasks {
		if task == nil {
			continue
		}

		wg.Add(1)
		go func(task *proto.Task) {
			defer wg.Done()

			p.predictTaskACLCalls(chaincode, task)
		}(task)
	}

	wg.Wait()

	requestBytes := make([][]byte, len(p.invocationBuffer))
	i := 0
	for _, bytes := range p.invocationBuffer {
		requestBytes[i] = bytes
		i++
	}

	_, err := helpers.GetAccountsInfo(stub, requestBytes)
	if err != nil {
		logger.Logger().Errorf("PredictAclCalls txID %s, failed to invoke acl calls: %v", stub.GetTxID(), err)
	}
}

func (p *predictACL) predictTaskACLCalls(chaincode *Chaincode, task *proto.Task) {
	var (
		method       = chaincode.Router().Method(task.GetMethod())
		authRequired = chaincode.Router().AuthRequired(method)
		args         = task.GetArgs()
		argCount     = chaincode.Router().ArgCount(method)
	)

	// get signers for evaluate *Sender
	signers := getSigners(authRequired, argCount, task)
	if signers != nil {
		p.addCall(helpers.FnCheckKeys, strings.Join(signers, "/"))
	}

	inputVal := reflect.ValueOf(chaincode.contract)
	methodVal := inputVal.MethodByName(method)
	if !methodVal.IsValid() {
		return
	}
	methodArgs := args
	if authRequired {
		methodArgs = methodArgs[3 : 3+argCount-1]
	}
	methodType := methodVal.Type()

	// check method input args without signer, to skip signers in future for
	if (authRequired && methodType.NumIn()-1 != len(methodArgs)) ||
		(!authRequired && methodType.NumIn() != len(methodArgs)) {
		return
	}
	if strings.Contains(task.GetMethod(), FnTransfer) && len(methodArgs) > 0 {
		p.addCall(helpers.FnCheckAddress, methodArgs[0])
	}

	for i, arg := range methodArgs {
		// skip sender from methodType args
		indexInputArg := i
		if authRequired {
			indexInputArg++
		}

		t := methodType.In(indexInputArg)
		if t.Kind() != reflect.Pointer {
			continue
		}

		argInterface := reflect.New(t.Elem()).Interface()
		_, ok := argInterface.(*types.Address)
		if !ok {
			continue
		}

		_, err := types.AddrFromBase58Check(arg)
		if err != nil {
			continue
		}
		p.addCall(helpers.FnGetAccountInfo, arg)
	}
}

func (p *predictACL) addCall(method string, arg string) {
	logger.Logger().Debugf("PredictAcl txID %s: adding acl call: method %s arg %s", p.stub.GetTxID(), method, arg)

	if len(arg) == 0 {
		return
	}

	key := method + arg

	p.mu.RLock()
	_, ok := p.invocationBuffer[key]
	p.mu.RUnlock()

	if !ok {
		p.mu.Lock()
		defer p.mu.Unlock()

		if _, ok = p.invocationBuffer[key]; !ok {
			bytes, err := json.Marshal([]string{method, arg})
			if err != nil {
				logger.Logger().Errorf(
					"PredictAcl txID %s: adding acl call: failed to marshal, method: '%s', arg '%s': %v",
					p.stub.GetTxID(),
					method,
					arg,
					err,
				)

				return
			}

			p.invocationBuffer[key] = bytes
		}
	}
}

func getSigners(authRequired bool, argCount int, task *proto.Task) []string {
	if !authRequired {
		return nil
	}

	invocation, err := parseInvocationDetails(argCount, task.GetArgs())
	if err != nil {
		return nil
	}

	return invocation.signatureArgs[:invocation.signersCount]
}
