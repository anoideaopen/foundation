package reflect

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// Error types.
var (
	ErrIncorrectArgumentCount = errors.New("incorrect number of arguments")
	ErrMethodNotFound         = errors.New("method not found")
)

// ValidateArguments validates the arguments for the specified method on the given value using reflection.
// It checks whether the specified method exists on the value 'v' and if the number of provided arguments matches
// the method's expected input parameters. Additionally, it attempts to convert the string arguments into the
// expected types. If an argument implements the types.Checker or types.CheckerWithStub
// interfaces, its Check method is called (with the provided stub if available).
//
// The function returns an error if the method is not found, the number of arguments is incorrect, or if an error
// occurs during argument conversion or validation.
func ValidateArguments(v any, method string, stub shim.ChaincodeStubInterface, args ...string) error {
	inputVal := reflect.ValueOf(v)

	methodVal := inputVal.MethodByName(method)
	if !methodVal.IsValid() {
		return fmt.Errorf("%w: %s", ErrMethodNotFound, method)
	}

	methodType := methodVal.Type()
	if methodType.NumIn() != len(args) {
		return fmt.Errorf(
			"%w: found %d but expected %d: validate %s",
			ErrIncorrectArgumentCount,
			len(args),
			methodType.NumIn(),
			method,
		)
	}

	for i, arg := range args {
		value, err := ParseValue(arg, methodType.In(i), stub)
		if err != nil {
			return fmt.Errorf("%w: validate %s, argument %d", err, method, i)
		}

		iface := value.Interface()

		if checker, ok := iface.(types.Checker); ok {
			if err := checker.Check(); err != nil {
				return fmt.Errorf(
					"%w: '%s': validation failed: '%v': validate %s, argument %d",
					ErrInvalidArgumentValue,
					arg,
					err.Error(),
					method,
					i,
				)
			}
		}

		if stub == nil {
			continue
		}
		if checker, ok := iface.(types.CheckerWithStub); ok {
			if err := checker.CheckWithStub(stub); err != nil {
				return fmt.Errorf(
					"%w: '%s': validation failed: '%v': validate %s, argument %d",
					ErrInvalidArgumentValue,
					arg,
					err.Error(),
					method,
					i,
				)
			}
		}
	}

	return nil
}
