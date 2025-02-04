package reflect

import (
	"fmt"
	"reflect"

	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
)

// Call invokes a specified method on a given value using reflection. The method to be invoked is
// identified by its name. It checks whether the specified method exists on the value 'v' and if
// the number of provided arguments matches the method's expected input parameters.
//
// The function returns a slice of any type representing the output from the called method, and an
// error if the method is not found, the number of arguments does not match, or if an error occurs
// during argument conversion or method invocation.
//
// Example:
//
//	type MyType struct {
//	    Data string
//	}
//
//	func (m *MyType) Update(data string) string {
//	    m.Data = data
//	    return fmt.Sprintf("Updated data to: %s", m.Data)
//	}
//
//	func main() {
//	    myInstance := &MyType{}
//	    output, err := Call(myInstance, "Update", nil, `"New data"`)
//	    if err != nil {
//	        log.Fatalf("Error invoking method: %v", err)
//	    }
//	    fmt.Println(output[0]) // Output: Updated data to: New data
//	}
func Call(v any, method string, stub shim.ChaincodeStubInterface, args ...string) ([]any, error) {
	inputVal := reflect.ValueOf(v)

	methodVal := inputVal.MethodByName(method)
	if !methodVal.IsValid() {
		return nil, fmt.Errorf("%w: %s", ErrMethodNotFound, method)
	}

	methodType := methodVal.Type()
	if methodType.NumIn() != len(args) {
		return nil, fmt.Errorf(
			"%w: found %d but expected %d: call %s",
			ErrIncorrectArgumentCount,
			len(args),
			methodType.NumIn(),
			method,
		)
	}

	var (
		in  = make([]reflect.Value, len(args))
		err error
	)
	for i, arg := range args {
		if in[i], err = ParseValue(arg, methodType.In(i), stub); err != nil {
			return nil, fmt.Errorf("%w: call %s, argument %d", err, method, i)
		}
	}

	output := make([]any, methodType.NumOut())
	for i, res := range methodVal.Call(in) {
		output[i] = res.Interface()
	}

	return output, nil
}
