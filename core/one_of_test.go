package core

import "testing"

func TestOneOf(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		ss       []string
		expected bool
	}{
		{
			name:     "Present in list",
			s:        "apple",
			ss:       []string{"banana", "apple", "orange"},
			expected: true,
		},
		{
			name:     "Not present in list",
			s:        "grape",
			ss:       []string{"banana", "apple", "orange"},
			expected: false,
		},
		{
			name:     "Empty list",
			s:        "apple",
			ss:       []string{},
			expected: false,
		},
		{
			name:     "Empty string search",
			s:        "",
			ss:       []string{"banana", "apple", "orange"},
			expected: false,
		},
		{
			name:     "List with empty string",
			s:        "apple",
			ss:       []string{"banana", "", "orange"},
			expected: false,
		},
		{
			name:     "Searching for empty string present",
			s:        "",
			ss:       []string{"banana", "", "orange"},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if result := OneOf(tc.s, tc.ss...); result != tc.expected {
				t.Errorf("Test %s failed: expected %v, got %v", tc.name, tc.expected, result)
			}
		})
	}
}
