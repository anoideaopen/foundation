package stringsx

import "testing"

func TestTrimFirstPrefix(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		prefixes []string
		expected string
	}{
		{
			name:     "Remove first matching prefix",
			s:        "unexample",
			prefixes: []string{"pre", "un"},
			expected: "example",
		},
		{
			name:     "No prefix matched",
			s:        "example",
			prefixes: []string{"pre", "un"},
			expected: "example",
		},
		{
			name:     "Multiple matches, only first removed",
			s:        "ununexample",
			prefixes: []string{"un"},
			expected: "unexample",
		},
		{
			name:     "Empty string",
			s:        "",
			prefixes: []string{"pre", "un"},
			expected: "",
		},
		{
			name:     "Empty prefix list",
			s:        "unexample",
			prefixes: []string{},
			expected: "unexample",
		},
		{
			name:     "Prefix list with empty string",
			s:        "example",
			prefixes: []string{"", "ex"},
			expected: "ample",
		},
		{
			name:     "String with unicode characters",
			s:        "ñandúexample",
			prefixes: []string{"ñandú"},
			expected: "example",
		},
		{
			name:     "Prefix longer than the string",
			s:        "ex",
			prefixes: []string{"example"},
			expected: "ex",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if result := TrimFirstPrefix(tc.s, tc.prefixes...); result != tc.expected {
				t.Errorf("Failed %s: expected '%s', got '%s'", tc.name, tc.expected, result)
			}
		})
	}
}
