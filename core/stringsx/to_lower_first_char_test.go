package stringsx

import "testing"

func TestLowerFirstChar(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard case conversion",
			input:    "Hello",
			expected: "hello",
		},
		{
			name:     "Already lowercase",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Unicode upper to lower",
			input:    "Éclair",
			expected: "éclair",
		},
		{
			name:     "Number leading",
			input:    "1stPlace",
			expected: "1stPlace",
		},
		{
			name:     "Single uppercase letter",
			input:    "A",
			expected: "a",
		},
		{
			name:     "Single lowercase letter",
			input:    "a",
			expected: "a",
		},
		{
			name:     "All uppercase",
			input:    "ABC",
			expected: "aBC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if result := LowerFirstChar(tc.input); result != tc.expected {
				t.Errorf("Test %s failed: expected '%s', got '%s'", tc.name, tc.expected, result)
			}
		})
	}
}
