package routing

import (
	"errors"
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/core/reflectx"
	"github.com/anoideaopen/foundation/core/stringsx"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

const (
	batchedTransactionPrefix      = "Tx"
	transactionWithoutBatchPrefix = "NBTx"
	queryTransactionPrefix        = "Query"
)

var ErrMethodAlreadyDefined = errors.New("pure method has already defined")

type ReflectRouter struct {
	contract any
	cache    map[string]Endpoint
}

func NewReflectRouter(contract any) (*ReflectRouter, error) {
	r := &ReflectRouter{
		contract: contract,
	}

	allowedMethodPrefixes := []string{
		batchedTransactionPrefix,
		transactionWithoutBatchPrefix,
		queryTransactionPrefix,
	}

	methods := reflectx.Methods(contract)

	duplicates := make(map[string]struct{})
	for _, method := range methods {
		if !stringsx.HasPrefix(method, allowedMethodPrefixes...) {
			continue
		}

		method = stringsx.TrimFirstPrefix(method, allowedMethodPrefixes...)
		method = stringsx.LowerFirstChar(method)

		if _, ok := duplicates[method]; ok {
			return nil, fmt.Errorf("%w, method: '%s'", ErrMethodAlreadyDefined, method)
		}

		duplicates[method] = struct{}{}
	}

	return r, nil
}

func (r *ReflectRouter) Endpoints() []Endpoint {
	if r.cache != nil {
		endpoints := make([]Endpoint, 0, len(r.cache))

		for _, ep := range r.cache {
			endpoints = append(endpoints, ep)
		}

		return endpoints
	}

	r.cache = make(map[string]Endpoint)
	endpoints := []Endpoint{}
	for _, method := range reflectx.Methods(r.contract) {
		var (
			chaincodeFunction string
			endpointType      EndpointType
		)
		switch {
		case strings.HasPrefix(method, batchedTransactionPrefix):
			endpointType = EndpointTypeTransaction
			chaincodeFunction = strings.TrimPrefix(method, batchedTransactionPrefix)

		case strings.HasPrefix(method, transactionWithoutBatchPrefix):
			endpointType = EndpointTypeInvoke
			chaincodeFunction = strings.TrimPrefix(method, transactionWithoutBatchPrefix)

		case strings.HasPrefix(method, queryTransactionPrefix):
			endpointType = EndpointTypeQuery
			chaincodeFunction = strings.TrimPrefix(method, queryTransactionPrefix)
		default:
			continue
		}

		if len(chaincodeFunction) == 0 {
			continue
		}

		in, out := reflectx.InOut(r.contract, method)
		chaincodeFunction = stringsx.LowerFirstChar(chaincodeFunction)

		ep := Endpoint{
			Type:          endpointType,
			ChaincodeFunc: chaincodeFunction,
			MethodName:    method,
			NumArgs:       in,
			NumReturns:    out,
		}

		endpoints = append(endpoints, ep)
		r.cache[chaincodeFunction] = ep
	}

	return endpoints
}

func (r *ReflectRouter) Endpoint(chaincodeFunc string) (Endpoint, error) {
	if r.cache == nil {
		_ = r.Endpoints()
	}

	if ep, ok := r.cache[chaincodeFunc]; ok {
		return ep, nil
	}

	return Endpoint{}, fmt.Errorf("method '%s' not found", chaincodeFunc)
}

func (r *ReflectRouter) ValidateArguments(
	method string,
	stub shim.ChaincodeStubInterface,
	args ...string,
) error {
	return reflectx.ValidateArguments(r.contract, method, stub, args...)
}

func (r *ReflectRouter) Call(
	method string,
	stub shim.ChaincodeStubInterface,
	args ...string,
) ([]any, error) {
	if stubSetter, ok := r.contract.(interface {
		SetStub(shim.ChaincodeStubInterface)
	}); ok {
		stubSetter.SetStub(stub)
	}

	return reflectx.Call(r.contract, method, args...)
}
