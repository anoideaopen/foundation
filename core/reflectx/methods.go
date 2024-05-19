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
