package reflectx

import (
	"reflect"
	"testing"
)

// Define a struct with methods
type TestStructForMethods struct{}

func (TestStructForMethods) MethodOne() {}
func (TestStructForMethods) MethodTwo() {}

// Define an empty struct
type EmptyStructForMethods struct{}

// Define a pointer receiver method
type PointerStructForMethods struct{}

func (*PointerStructForMethods) PtrMethod() {}

// Define an interface
type MyInterfaceForMethods interface {
	InterfaceMethod()
}

// Method for testing interfaces
type InterfaceImplForMethods struct{}

func (InterfaceImplForMethods) InterfaceMethod() {}

func TestMethods(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected []string
	}{
		{
			name:     "struct with methods",
			input:    TestStructForMethods{},
			expected: []string{"MethodOne", "MethodTwo"},
		},
		{
			name:     "empty struct",
			input:    EmptyStructForMethods{},
			expected: []string{},
		},
		{
			name:     "pointer to struct with method",
			input:    &PointerStructForMethods{},
			expected: []string{"PtrMethod"},
		},
		{
			name:     "interface implementation",
			input:    InterfaceImplForMethods{},
			expected: []string{"InterfaceMethod"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Methods(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Test %s failed: expected %v, got %v", tc.name, tc.expected, result)
			}
		})
	}
}
