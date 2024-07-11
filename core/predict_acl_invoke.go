package core

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"

	"github.com/anoideaopen/foundation/core/helpers"
	"github.com/anoideaopen/foundation/core/logger"
	"github.com/anoideaopen/foundation/core/routing"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

const (
	FnTransfer = "transfer"
)

type predictACL struct {
	stub     shim.ChaincodeStubInterface
	m        sync.RWMutex
	callsMap map[string][]byte
}

func predictACLCalls(stub shim.ChaincodeStubInterface, tasks []*proto.Task, chaincode *Chaincode) {
	methods := chaincode.Router().Methods()
	p := predictACL{
		stub:     stub,
		m:        sync.RWMutex{},
		callsMap: make(map[string][]byte),
	}
	wg := &sync.WaitGroup{}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		wg.Add(1)
		go func(task *proto.Task) {
			defer wg.Done()
			method := methods[task.GetMethod()]
			p.predictTaskACLCalls(chaincode, task, method)
		}(task)
	}
	wg.Wait()

	requestBytes := make([][]byte, len(p.callsMap))
	i := 0
	for _, bytes := range p.callsMap {
		requestBytes[i] = bytes
		i++
	}

	// TODO: need to add retry if error cause is network error
	_, err := helpers.GetAccountsInfo(stub, requestBytes)
	if err != nil {
		logger.Logger().Errorf("PredictAclCalls txID %s, failed to invoke acl calls: %v", stub.GetTxID(), err)
	}
}

func (p *predictACL) predictTaskACLCalls(chaincode *Chaincode, task *proto.Task, method routing.Method) {
	signers := getSigners(method, task)
	if signers != nil {
		p.addCall(helpers.FnCheckKeys, strings.Join(signers, "/"))
	}

	inputVal := reflect.ValueOf(chaincode.contract)
	methodVal := inputVal.MethodByName(method.Method)
	if !methodVal.IsValid() {
		return
	}

	methodArgs := task.GetArgs()[3 : 3+(method.ArgCount-1)]
	methodType := methodVal.Type()

	// check method input args without signer, to skip signers in future for
	lenSigners := len(signers)
	if methodType.NumIn()-lenSigners != len(methodArgs) {
		return
	}
	if strings.Contains(task.GetMethod(), FnTransfer) && len(methodArgs) > 0 {
		p.addCall(helpers.FnCheckAddress, methodArgs[0])
	}

	for i, arg := range methodArgs {
		// skip signers from methodType args
		indexInputArg := i + lenSigners
		if indexInputArg > methodType.NumIn() {
			continue
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
	p.m.RLock()
	_, ok := p.callsMap[key]
	p.m.RUnlock()
	if !ok {
		p.m.Lock()
		defer p.m.Unlock()
		_, ok = p.callsMap[key]
		if !ok {
			bytes, err := json.Marshal([]string{method, arg})
			if err != nil {
				logger.Logger().Errorf("PredictAcl txID %s: adding acl call: failed to marshal, method: '%s', arg '%s': %v",
					p.stub.GetTxID(), method, arg, err)
				return
			}
			p.callsMap[key] = bytes
		}
	}
}

func getSigners(method routing.Method, task *proto.Task) []string {
	if !method.AuthRequired {
		return nil
	}

	invocation, err := parseInvocationDetails(method, task.GetArgs())
	if err != nil {
		return nil
	}

	return invocation.signatureArgs[:invocation.signersCount]
}
