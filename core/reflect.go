package core

import (
	"errors"
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/core/reflectx"
	"github.com/anoideaopen/foundation/core/routing"
	stringsx "github.com/anoideaopen/foundation/core/stringsx"
	"github.com/anoideaopen/foundation/core/types"
)

var (
	ErrMethodAlreadyDefined = errors.New("pure method has already defined")
	ErrUnsupportedMethod    = errors.New("unsupported method")
	ErrInvalidMethodName    = errors.New("invalid method name")
)

func NewEndpoint(name string, of any) (*routing.Endpoint, error) {
	ep := &routing.Endpoint{
		Type:          0,
		ChaincodeFunc: "",
		MethodName:    name,
		NumArgs:       0,
		NumReturns:    0,
		RequiresAuth:  false,
	}

	switch {
	case strings.HasPrefix(ep.MethodName, batchedTransactionPrefix):
		ep.Type = routing.EndpointTypeTransaction
		ep.ChaincodeFunc = strings.TrimPrefix(ep.MethodName, batchedTransactionPrefix)

	case strings.HasPrefix(ep.MethodName, transactionWithoutBatchPrefix):
		ep.Type = routing.EndpointTypeInvoke
		ep.ChaincodeFunc = strings.TrimPrefix(ep.MethodName, transactionWithoutBatchPrefix)

	case strings.HasPrefix(ep.MethodName, queryTransactionPrefix):
		ep.Type = routing.EndpointTypeQuery
		ep.ChaincodeFunc = strings.TrimPrefix(ep.MethodName, queryTransactionPrefix)

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedMethod, ep.MethodName)
	}

	if len(ep.ChaincodeFunc) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMethodName, ep.MethodName)
	}

	ep.ChaincodeFunc = stringsx.LowerFirstChar(ep.ChaincodeFunc)
	ep.NumArgs, ep.NumReturns = reflectx.MethodParamCounts(of, ep.MethodName)
	ep.RequiresAuth = reflectx.IsArgOfType(of, ep.MethodName, 0, &types.Sender{})

	return ep, nil
}

var (
	swapMethods      = []string{"QuerySwapGet", "TxSwapBegin", "TxSwapCancel"}
	multiSwapMethods = []string{"QueryMultiSwapGet", "TxMultiSwapBegin", "TxMultiSwapCancel"}
)

func parseContractEndpoints(in BaseContractInterface) (map[string]*routing.Endpoint, error) {
	cfgOptions := in.ContractConfig().GetOptions()

	swapsDisabled := cfgOptions.GetDisableSwaps()
	multiswapsDisabled := cfgOptions.GetDisableMultiSwaps()
	disabledMethods := cfgOptions.GetDisabledFunctions()

	out := make(map[string]*routing.Endpoint)
	for _, method := range reflectx.Methods(in) {
		if stringsx.OneOf(method, disabledMethods...) ||
			(swapsDisabled && stringsx.OneOf(method, swapMethods...)) ||
			(multiswapsDisabled && stringsx.OneOf(method, multiSwapMethods...)) {
			continue
		}

		ep, err := NewEndpoint(method, in)
		if err != nil {
			if errors.Is(err, ErrUnsupportedMethod) {
				continue
			}

			return nil, err
		}

		out[ep.ChaincodeFunc] = ep
	}

	return out, nil
}
