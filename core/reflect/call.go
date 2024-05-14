package reflect

import (
	"encoding"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Error types.
var (
	ErrIncorrectArgumentCount  = errors.New("incorrect number of arguments")
	ErrInvalidArgumentValue    = errors.New("invalid argument value")
	ErrMethodNotFound          = errors.New("method not found")
	ErrUnsupportedArgumentType = errors.New("unsupported argument type")
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
			return nil, fmt.Errorf("%w: call %s", err, method)
		}
	}

	output := make([]any, methodType.NumOut())
	for i, res := range methodVal.Call(in) {
		output[i] = res.Interface()
	}

	return output, nil
}

func valueOf(s string, t reflect.Type) (reflect.Value, error) {
	var (
		argRaw     = []byte(s)
		argType    = t
		argPointer = t.Kind() == reflect.Pointer
	)

	// If the type is a pointer, create a new instance of the element type
	// and assign it to the pointer. Otherwise, create a new instance of the
	// type and assign it to the element of the newly created pointer.
	// This is necessary because we can't call json.Unmarshal directly on a
	// pointer.
	var (
		argValue reflect.Value
		outValue reflect.Value
	)
	if argPointer {
		argValue = reflect.New(t.Elem()) // create a new instance of the element type
		outValue = argValue              // assign the pointer to the return value
	} else {
		argValue = reflect.New(t)  // create a new instance of the type
		outValue = argValue.Elem() // assign the element of the pointer to the return value
	}

	// Trying to check if the argument type is a string or *string.
	switch {
	case argType.Kind() == reflect.String:
		outValue.SetString(string(argRaw))
		return outValue, nil

	case argPointer && argType.Elem().Kind() == reflect.String:
		argValue.Elem().SetString(string(argRaw))
		return outValue, nil
	}

	argInterface := argValue.Interface()

	// Check if the argument is a valid json string.
	if json.Valid(argRaw) {
		var err error
		if protoMessage, ok := argInterface.(proto.Message); ok {
			err = protojson.Unmarshal(argRaw, protoMessage)
		} else {
			err = json.Unmarshal(argRaw, argInterface)
		}
		if err == nil {
			return outValue, nil
		}
	}

	// Trying to use the encoding.TextUnmarshaler interface.
	if unmarshaler, ok := argInterface.(encoding.TextUnmarshaler); ok {
		if err := unmarshaler.UnmarshalText(argRaw); err == nil {
			return outValue, nil
		}
	}

	// Trying to use the proto.Message interface.
	if protoMessage, ok := argInterface.(proto.Message); ok {
		if err := proto.Unmarshal(argRaw, protoMessage); err == nil {
			return outValue, nil
		}
	}

	// Trying to use the encoding.BinaryUnmarshaler interface.
	if unmarshaler, ok := argInterface.(encoding.BinaryUnmarshaler); ok {
		if err := unmarshaler.UnmarshalBinary(argRaw); err == nil {
			return outValue, nil
		}
	}

	// Trying to use the gob.GobDecoder interface.
	if decoder, ok := argInterface.(gob.GobDecoder); ok {
		if err := decoder.GobDecode(argRaw); err == nil {
			return outValue, nil
		}
	}

	// Unsupported type.
	return outValue, fmt.Errorf("%w: type %s", ErrUnsupportedArgumentType, argType.String())
}
