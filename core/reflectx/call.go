package reflectx

import (
	"errors"
	"fmt"
	"reflect"
)

// Error types.
var (
	ErrIncorrectArgumentCount = errors.New("incorrect number of arguments")
	ErrInvalidArgumentValue   = errors.New("invalid argument value")
	ErrMethodNotFound         = errors.New("method not found")
)

// Call invokes a specified method on a given value using reflection. The method to be invoked is identified by its name.
// It checks whether the specified method exists on the value 'v' and if the number of provided arguments matches the
// method's expected input parameters. If the method is found and the arguments match, it attempts to convert
// the string arguments into the method's expected argument types using various unmarshalers or decoders, such as JSON,
// proto.Message, encoding.TextUnmarshaler, encoding.BinaryUnmarshaler, and gob.GobDecoder.
//
// The process follows these steps:
//  1. Check if the input string arguments are valid JSON; if so, attempt to unmarshal into the expected types.
//  2. If not valid JSON, attempt to decode the string using the aforementioned interfaces in sequential order based on
//     the type compatibility.
//  3. Call the method with the prepared arguments and capture the output.
//
// The function returns a slice of any type representing the output from the called method, and an error if the method
// is not found, the number of arguments does not match, or if an error occurs during argument conversion or method invocation.
//
// Parameters:
//   - v: The value on which the method is to be invoked.
//   - method: The name of the method to invoke.
//   - args: A slice of strings representing the arguments for the method.
//
// Returns:
//   - []any: A slice containing the outputs of the method, or nil if an error occurs.
//   - error: An error if the method is not found, the number of arguments is incorrect, or any other issue during invocation.
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
//	    output, err := Call(myInstance, "Update", `"New data"`)
//	    if err != nil {
//	        log.Fatalf("Error invoking method: %v", err)
//	    }
//	    fmt.Println(output[0]) // Output: Updated data to: New data
//	}
func Call(v any, method string, args ...string) ([]any, error) {
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
		if in[i], err = valueOf(arg, methodType.In(i)); err != nil {
			return nil, fmt.Errorf("%w: call %s, argument %d", err, method, i)
		}
	}

	output := make([]any, methodType.NumOut())
	for i, res := range methodVal.Call(in) {
		output[i] = res.Interface()
	}

	return output, nil
}
