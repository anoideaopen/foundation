package core

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/anoideaopen/foundation/core/types"
)

var ErrMethodAlreadyDefined = errors.New("pure method has already defined")

var funcPrefixes = []string{txPrefix, queryPrefix, noBatchPrefix}

// In is a struct for input parameters
type In struct {
	kind          reflect.Type
	prepareToSave reflect.Value
	convertToCall reflect.Value
}

// Fn is a struct for function
type Fn struct {
	Name           string
	FName          string
	fn             reflect.Value
	query          bool
	noBatch        bool
	needsAuth      bool
	in             []In
	hasOutputValue bool
}

func (f *Fn) getInputs(method reflect.Method) error {
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

func checkConvertationMethod(method reflect.Method, in0, in1, in2, out0, out1 string) error {
	tp := method.Type
	if tp.In(0).String() != in0 || tp.In(1).String() != in1 ||
		tp.In(2).String() != in2 || tp.Out(0).String() != out0 || //nolint:gomnd
		tp.Out(1).String() != out1 {
		return fmt.Errorf("method %s can not be convertor", method.Name)
	}
	return nil
}

func checkOut(method reflect.Method) (bool, error) {
	count := method.Type.NumOut()
	if count == 1 && method.Type.Out(0).String() == "error" {
		return false, nil
	}
	if count == 2 && method.Type.Out(1).String() == "error" {
		return true, nil
	}
	return false, errors.New("unknown output types " + method.Name)
}

func toLowerFirstLetter(in string) string {
	return string(unicode.ToLower(rune(in[0]))) + in[1:]
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// validateContractMethods checks contract has duplicated method names with funcPrefixes.
func validateContractMethods(bci BaseContractInterface) error {
	t := reflect.TypeOf(bci)
	methods := make([]string, len(funcPrefixes))
	for methodIdx := 0; methodIdx < t.NumMethod(); methodIdx++ {
		mn := t.Method(methodIdx).Name

		for _, prefix := range funcPrefixes {
			if !strings.HasPrefix(mn, prefix) {
				continue
			}

			noPrefixFn := strings.TrimLeft(mn, prefix)

			// We need to figure out if it's worth looking for duplicates with private methods.
			methods = append(methods, mn)
			for _, p := range funcPrefixes {
				if p == prefix {
					continue
				}

				method := strings.Join([]string{p, noPrefixFn}, "")
				if _, found := t.MethodByName(method); found {
					methods = append(methods, method)
					sort.Strings(methods)
				}
			}

			if len(methods) > 1 {
				return fmt.Errorf(
					"failed, %w, method: '%s', cc methods: %v",
					ErrMethodAlreadyDefined,
					toLowerFirstLetter(noPrefixFn),
					methods,
				)
			}
		}

		methods = methods[:0]
	}

	return nil
}

func parseContractMethods(in BaseContractInterface) (ContractMethods, error) {
	cfgOptions := in.ContractConfig().GetOptions()

	out := make(map[string]*Fn)
	t := reflect.TypeOf(in)
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		nb := false
		query := false
		if contains(cfgOptions.GetDisabledFunctions(), method.Name) {
			continue
		}
		if cfgOptions.GetDisableSwaps() && (method.Name == "QuerySwapGet" ||
			method.Name == "TxSwapBegin" || method.Name == "TxSwapCancel") {
			continue
		}
		if cfgOptions.GetDisableMultiSwaps() && (method.Name == "QueryMultiSwapGet" ||
			method.Name == "TxMultiSwapBegin" || method.Name == "TxMultiSwapCancel") {
			continue
		}

		var methodNameTruncated string
		switch {
		case strings.HasPrefix(method.Name, txPrefix):
			methodNameTruncated = strings.TrimPrefix(method.Name, txPrefix)

		case strings.HasPrefix(method.Name, noBatchPrefix):
			nb = true
			methodNameTruncated = strings.TrimPrefix(method.Name, noBatchPrefix)

		case strings.HasPrefix(method.Name, queryPrefix):
			query = true
			nb = true
			methodNameTruncated = strings.TrimPrefix(method.Name, queryPrefix)

		default:
			continue
		}

		if len(methodNameTruncated) == 0 {
			continue
		}

		functionName := toLowerFirstLetter(methodNameTruncated) // example: QuerySwapGet => swapGet

		if _, ok := out[functionName]; ok {
			return nil, fmt.Errorf("%w, method: %s", ErrMethodAlreadyDefined, functionName)
		}

		out[functionName] = &Fn{
			Name:    method.Name,
			FName:   functionName,
			fn:      method.Func,
			noBatch: nb,
			query:   query,
		}

		err := out[functionName].getInputs(method)
		if err != nil {
			return nil, err
		}

		out[functionName].hasOutputValue, err = checkOut(method)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

type ContractMethods map[string]*Fn

func (cm *ContractMethods) Method(name string) (*Fn, error) {
	method, exists := (*cm)[name]
	if !exists {
		return nil, fmt.Errorf("method '%s' not found", name)
	}

	return method, nil
}
