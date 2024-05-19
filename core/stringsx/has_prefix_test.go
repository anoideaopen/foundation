package stringsx

import "testing"

func TestHasPrefix(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		prefixes []string
		expected bool
	}{
		{
			name:     "String has valid prefix",
			s:        "exampleString",
			prefixes: []string{"ex", "pre", "no"},
			expected: true,
		},
		{
			name:     "String does not have a valid prefix",
			s:        "testString",
			prefixes: []string{"ex", "pre", "no"},
			expected: false,
		},
		{
			name:     "Empty string and valid prefixes",
			s:        "",
			prefixes: []string{"ex", "pre", "no"},
			expected: false,
		},
		{
			name:     "Valid string and empty prefixes list",
			s:        "exampleString",
			prefixes: []string{},
			expected: false,
		},
		{
			name:     "Both string and prefixes list are empty",
			s:        "",
			prefixes: []string{},
			expected: false,
		},
		{
			name:     "Prefix exact match to string",
			s:        "hello",
			prefixes: []string{"hello"},
			expected: true,
		},
		{
			name:     "Multiple valid prefixes",
			s:        "helloWorld",
			prefixes: []string{"he", "hel", "hello"},
			expected: true,
		},
		{
			name:     "Unicode and special characters",
			s:        "ñandú",
			prefixes: []string{"ñ", "n", "ñan"},
			expected: true,
		},
		{
			name:     "Case sensitivity check",
			s:        "HelloWorld",
			prefixes: []string{"hello"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if result := HasPrefix(tc.s, tc.prefixes...); result != tc.expected {
				t.Errorf("Failed %s: expected %v, got %v", tc.name, tc.expected, result)
			}
		})
	}
}
