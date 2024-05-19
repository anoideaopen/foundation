package core

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/anoideaopen/foundation/core/reflectx"
	stringsx "github.com/anoideaopen/foundation/core/stringsx"
	"github.com/anoideaopen/foundation/core/types"
)

var (
	ErrMethodAlreadyDefined = errors.New("pure method has already defined")
	ErrUnsupportedMethod    = errors.New("unsupported method")
	ErrInvalidMethodName    = errors.New("invalid method name")
)

var allowedMethodPrefixes = []string{txPrefix, queryPrefix, noBatchPrefix}

// In is a struct for input parameters
type In struct {
	kind          reflect.Type
	prepareToSave reflect.Value
	convertToCall reflect.Value
}

// Method is a struct for function
type Method struct {
	Name           string
	FunctionName   string
	query          bool
	noBatch        bool
	needsAuth      bool
	in             []In
	hasOutputValue bool
}

func NewMethod(name string, of any) (*Method, error) {
	m := &Method{
		Name:           name,
		FunctionName:   "",
		query:          false,
		noBatch:        false,
		needsAuth:      false,
		in:             []In{},
		hasOutputValue: false,
	}

	switch {
	case strings.HasPrefix(m.Name, txPrefix):
		m.FunctionName = strings.TrimPrefix(m.Name, txPrefix)

	case strings.HasPrefix(m.Name, noBatchPrefix):
		m.noBatch = true
		m.FunctionName = strings.TrimPrefix(m.Name, noBatchPrefix)

	case strings.HasPrefix(m.Name, queryPrefix):
		m.query = true
		m.noBatch = true
		m.FunctionName = strings.TrimPrefix(m.Name, queryPrefix)
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

	count := method.Type.NumIn()
	begin := 1
	if method.Type.NumIn() > 1 && method.Type.In(1).String() == "*types.Sender" {
		f.needsAuth = true
		begin = 2
	}
	f.in = make([]In, 0, count-1)
	for j := begin; j < count; j++ {
		inType := method.Type.In(j).String()

		in := In{kind: method.Type.In(j)}

		if m, ok := types.BaseTypes[inType]; ok {
			r := reflect.ValueOf(m)
			in.convertToCall = r
			f.in = append(f.in, in)
			continue
		}

		m, ok := method.Type.In(j).MethodByName("ConvertToCall")
		if !ok {
			return fmt.Errorf("unknown type: %s in method %s", method.Type.In(j).String(), method.Name)
		}
		if err := checkConvertationMethod(m, inType, "shim.ChaincodeStubInterface", "string", inType, "error"); err != nil {
			return err
		}
		in.convertToCall = m.Func

		if m, ok = method.Type.In(j).MethodByName("PrepareToSave"); ok {
			if err := checkConvertationMethod(m, inType, "shim.ChaincodeStubInterface", "string", "string", "error"); err != nil {
				return err
			}
			in.prepareToSave = m.Func
		}
		f.in = append(f.in, in)
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

func checkConvertationMethod(method reflect.Method, in0, in1, in2, out0, out1 string) error {
	tp := method.Type
	if tp.In(0).String() != in0 || tp.In(1).String() != in1 ||
		tp.In(2).String() != in2 || tp.Out(0).String() != out0 || //nolint:gomnd
		tp.Out(1).String() != out1 {
		return fmt.Errorf("method %s can not be convertor", method.Name)
	}
	return nil
}

// validateContractMethods checks contract has duplicated method names with funcPrefixes.
func validateContractMethods(bci BaseContractInterface) error {
	methods := reflectx.Methods(bci)

	duplicates := make(map[string]struct{})
	for _, method := range methods {
		if !stringsx.HasPrefix(method, allowedMethodPrefixes...) {
			continue
		}

		method = stringsx.TrimFirstPrefix(method, allowedMethodPrefixes...)
		method = stringsx.LowerFirstChar(method)

		if _, ok := duplicates[method]; ok {
			return fmt.Errorf("%w, method: '%s'", ErrMethodAlreadyDefined, method)
		}

		duplicates[method] = struct{}{}
	}

	return nil
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
