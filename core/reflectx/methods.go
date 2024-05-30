package reflectx

import (
	"reflect"
	"sort"
)

// Methods inspects the type of the given value 'v' using reflection and returns a slice of strings
// containing the names of all methods that are defined on its type. This function only considers
// exported methods (those starting with an uppercase letter) due to Go's visibility rules in reflection.
//
// Parameters:
//   - v: The value whose type's methods are to be listed.
//
// Returns:
//   - []string: A slice containing the names of all methods associated with the type of 'v'.
func Methods(v any) []string {
	methodNames := make([]string, 0)

	t := reflect.TypeOf(v)
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		methodNames = append(methodNames, method.Name)
	}

	sort.Strings(methodNames)

	return methodNames
}

// MethodParamCounts inspects the type of the given value 'v' and returns the number of input and output parameters
// of the specified method using reflection. If the method does not exist, it returns -1 for both values.
//
// Parameters:
//   - v: The value whose method's parameters are to be inspected.
//   - method: The name of the method to inspect.
//
// Returns:
//   - int: The number of input parameters of the method.
//   - int: The number of output parameters of the method.
func MethodParamCounts(v any, method string) (in int, out int) {
	inputVal := reflect.ValueOf(v)

	methodVal := inputVal.MethodByName(method)
	if !methodVal.IsValid() {
		return -1, -1
	}

	methodType := methodVal.Type()
	return methodType.NumIn(), methodType.NumOut()
}

// IsArgOfType checks if the i-th argument of the specified method on value 'v' is of the given type 'argType'.
//
// Parameters:
//   - v: The value whose method's argument is to be checked.
//   - method: The name of the method to inspect.
//   - i: The index of the argument to check (0-based).
//   - argType: An example value of the desired type.
//
// Returns:
//   - bool: True if the i-th argument is of the specified type, false otherwise.
func IsArgOfType(v any, method string, i int, argType any) bool {
	inputVal := reflect.ValueOf(v)

	methodVal := inputVal.MethodByName(method)
	if !methodVal.IsValid() {
		return false
	}

	methodType := methodVal.Type()
	if i < 0 || i >= methodType.NumIn() {
		return false
	}

	expectedType := reflect.TypeOf(argType)
	return methodType.In(i) == expectedType
}

// MethodReturnsError checks if the last return value of the specified method on value 'v' is of type error.
//
// Parameters:
//   - v: The value whose method's return value is to be checked.
//   - method: The name of the method to inspect.
//
// Returns:
//   - bool: True if the last return value is of type error, false otherwise.
func MethodReturnsError(v any, method string) bool {
	inputVal := reflect.ValueOf(v)

	methodVal := inputVal.MethodByName(method)
	if !methodVal.IsValid() {
		return false
	}

	methodType := methodVal.Type()
	numOut := methodType.NumOut()
	if numOut == 0 {
		return false
	}

	return methodType.Out(numOut-1) == reflect.TypeOf((*error)(nil)).Elem()
}
