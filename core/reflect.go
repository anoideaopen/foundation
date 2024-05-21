package core

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/anoideaopen/foundation/core/reflectx"
	stringsx "github.com/anoideaopen/foundation/core/stringsx"
)

var (
	ErrMethodAlreadyDefined = errors.New("pure method has already defined")
	ErrUnsupportedMethod    = errors.New("unsupported method")
	ErrInvalidMethodName    = errors.New("invalid method name")
)

// Method is a struct for function
type Method struct {
	Name           string
	FunctionName   string
	query          bool
	noBatch        bool
	needsAuth      bool
	hasOutputValue bool
	in             int
}

func NewMethod(name string, of any) (*Method, error) {
	m := &Method{
		Name:           name,
		FunctionName:   "",
		query:          false,
		noBatch:        false,
		needsAuth:      false,
		in:             0,
		hasOutputValue: false,
	}

	switch {
	case strings.HasPrefix(m.Name, batchedTransactionPrefix):
		m.FunctionName = strings.TrimPrefix(m.Name, batchedTransactionPrefix)

	case strings.HasPrefix(m.Name, transactionWithoutBatchPrefix):
		m.noBatch = true
		m.FunctionName = strings.TrimPrefix(m.Name, transactionWithoutBatchPrefix)

	case strings.HasPrefix(m.Name, queryTransactionPrefix):
		m.query = true
		m.noBatch = true
		m.FunctionName = strings.TrimPrefix(m.Name, queryTransactionPrefix)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedMethod, m.Name)
	}

	if len(m.FunctionName) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMethodName, m.Name)
	}

	if err := m.parseInput(of); err != nil {
		return nil, err
	}

	if err := m.parseOutput(of); err != nil {
		return nil, err
	}

	m.FunctionName = stringsx.LowerFirstChar(m.FunctionName)

	return m, nil
}

func (f *Method) parseInput(of any) error {
	t := reflect.TypeOf(of)

	method, ok := t.MethodByName(f.Name)
	if !ok {
		return fmt.Errorf("method '%s' not found", f.Name)
	}

	f.in = method.Type.NumIn() - 1 // WTF? TODO: fix this
	if method.Type.NumIn() > 1 && method.Type.In(1).String() == "*types.Sender" {
		f.needsAuth = true
		f.in--
	}

	return nil
}

func (f *Method) parseOutput(of any) error {
	t := reflect.TypeOf(of)

	method, ok := t.MethodByName(f.Name)
	if !ok {
		return fmt.Errorf("method '%s' not found", f.Name)
	}

	count := method.Type.NumOut()
	if count == 1 && method.Type.Out(0).String() == "error" {
		f.hasOutputValue = false
		return nil
	}

	if count == 2 && method.Type.Out(1).String() == "error" {
		f.hasOutputValue = true
		return nil
	}

	return errors.New("unknown output types " + method.Name)
}

var (
	swapMethods      = []string{"QuerySwapGet", "TxSwapBegin", "TxSwapCancel"}
	multiSwapMethods = []string{"QueryMultiSwapGet", "TxMultiSwapBegin", "TxMultiSwapCancel"}
)

func parseContractMethods(in BaseContractInterface) (map[string]*Method, error) {
	cfgOptions := in.ContractConfig().GetOptions()

	swapsDisabled := cfgOptions.GetDisableSwaps()
	multiswapsDisabled := cfgOptions.GetDisableMultiSwaps()
	disabledMethods := cfgOptions.GetDisabledFunctions()

	out := make(map[string]*Method)
	for _, method := range reflectx.Methods(in) {
		if stringsx.OneOf(method, disabledMethods...) ||
			(swapsDisabled && stringsx.OneOf(method, swapMethods...)) ||
			(multiswapsDisabled && stringsx.OneOf(method, multiSwapMethods...)) {
			continue
		}

		m, err := NewMethod(method, in)
		if err != nil {
			if errors.Is(err, ErrUnsupportedMethod) {
				continue
			}

			return nil, err
		}

		out[m.FunctionName] = m
	}

	return out, nil
}
