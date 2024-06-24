package reflectx

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// valueOf converts a string representation of an argument to a reflect.Value of the specified type.
// It attempts to unmarshal the string into the appropriate type using various methods such as JSON,
// encoding.TextUnmarshaler, encoding.BinaryUnmarshaler and codec.BytesDecoder.
//
// Parameters:
//   - s: The string representation of the argument.
//   - t: The reflect.Type to which the argument should be converted.
//
// Returns:
//   - reflect.Value: The converted value.
//   - error: An error if the conversion fails or the type is unsupported.
//
// The function follows these steps:
//  1. Checks if the target type is a string or a pointer to a string and handles these cases directly.
//  2. Attempts to unmarshal the string using the BytesDecoder or StubBytesDecoder code interface if implemented.
//  3. Attempts to unmarshal the string as JSON if it is valid JSON. Note that simple values such as numbers,
//     booleans, and null are also valid JSON if they are represented as strings.
//  4. Attempts to unmarshal the string using the encoding.TextUnmarshaler interface if implemented.
//  5. Attempts to unmarshal the string using the encoding.BinaryUnmarshaler interface if implemented.
//  6. Returns an error if none of the above methods succeed.
func valueOf(s string, t reflect.Type, stub shim.ChaincodeStubInterface) (reflect.Value, error) {
	argRaw := []byte(s)
	argPointer := t.Kind() == reflect.Pointer

	var (
		argValue reflect.Value
		outValue reflect.Value
	)
	if argPointer {
		argValue = reflect.New(t.Elem())
		outValue = argValue
	} else {
		argValue = reflect.New(t)
		outValue = argValue.Elem()
	}

	switch {
	case t.Kind() == reflect.String:
		outValue.SetString(string(argRaw))
		return outValue, nil
	case argPointer && t.Elem().Kind() == reflect.String:
		argValue.Elem().SetString(string(argRaw))
		return outValue, nil
	}

	argInterface := argValue.Interface()

	if decoder, ok := argInterface.(BytesDecoder); ok {
		if err := decoder.DecodeFromBytes(argRaw); err != nil {
			return outValue, NewValueError(s, t, err)
		}

		return outValue, nil
	}

	if decoder, ok := argInterface.(StubBytesDecoder); ok && stub != nil {
		if err := decoder.DecodeFromBytesWithStub(stub, argRaw); err != nil {
			return outValue, NewValueError(s, t, err)
		}

		return outValue, nil
	}

	if json.Valid(argRaw) {
		if err := json.Unmarshal(argRaw, argInterface); err != nil {
			return outValue, NewValueError(s, t, err)
		}

		return outValue, nil
	}

	if unmarshaler, ok := argInterface.(encoding.TextUnmarshaler); ok && utf8.ValidString(string(argRaw)) {
		if err := unmarshaler.UnmarshalText(argRaw); err != nil {
			return outValue, NewValueError(s, t, err)
		}

		return outValue, nil
	}

	if unmarshaler, ok := argInterface.(encoding.BinaryUnmarshaler); ok {
		if err := unmarshaler.UnmarshalBinary(argRaw); err != nil {
			return outValue, NewValueError(s, t, err)
		}

		return outValue, nil
	}

	return outValue, NewValueError(s, t, nil)
}

// ValueError is a custom error type that wraps both external and internal errors,
// providing additional context about the argument and the target type involved in the error.
type ValueError struct {
	external error
	internal error
	arg, t   string
}

// Error returns a formatted error message indicating the conversion failure.
// If an external error is present, it is included in the message.
//
// Returns:
//   - string: A formatted error message.
func (e ValueError) Error() string {
	if e.external == nil {
		return fmt.Sprintf("%v: '%s': for type '%s'", e.internal, e.arg, e.t)
	}

	return fmt.Sprintf("%v: '%s': for type '%s': '%v'", e.internal, e.arg, e.t, e.external)
}

// Is checks if the target error matches the internal error.
//
// Parameters:
//   - target: The target error to compare against.
//
// Returns:
//   - bool: True if the target error matches the internal error, false otherwise.
func (e ValueError) Is(target error) bool {
	return e.internal == target
}

// Unwrap returns the external error, if any.
//
// Returns:
//   - error: The external error, or nil if none is present.
func (e ValueError) Unwrap() error {
	return e.external
}

// NewValueError constructs an error message for invalid argument value conversions.
// Parameters:
//   - arg: The argument value as a string.
//   - t: The reflect.Type to which the argument was attempted to be converted.
//   - errOrNil: The error encountered during the conversion, if any.
//
// Returns:
//   - error: A formatted error message indicating the conversion failure.
func NewValueError(arg string, t reflect.Type, errOrNil error) error {
	return ValueError{
		external: errOrNil,
		internal: ErrInvalidArgumentValue,
		arg:      arg,
		t:        t.String(),
	}
}
