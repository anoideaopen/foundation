package reflect

import (
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"
)

// Methods inspects the type of the given value 'v' using reflection and returns a slice of strings
// containing the names of all methods that are defined on its type. This function only considers
// exported methods (those starting with an uppercase letter) due to Go's visibility rules in reflection.
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

// IsArgOfType checks if the i-th argument of the specified method
// on value 'v' is of the given type 'argType'.
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

// InputParamCounts inspects the type of the given value 'v' and returns the number of
// input parameters of the specified method using reflection. If the method does not exist,
// it returns -1.
func InputParamCounts(v any, method string) (in int) {
	var (
		inputVal  = reflect.ValueOf(v)
		methodVal = inputVal.MethodByName(method)
	)
	if !methodVal.IsValid() {
		return -1
	}

	return methodVal.Type().NumIn()
}

// LowerFirstChar takes a string and returns a new string with the first character converted
// to lowercase.
func LowerFirstChar(s string) string {
	if s == "" {
		return ""
	}

	firstRune, size := utf8.DecodeRuneInString(s)
	lowerFirstRune := strings.ToLower(string(firstRune))

	return lowerFirstRune + s[size:]
}

// MethodReturnsError checks if the last return value of the specified method on value 'v' is of type error.
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
