package stringsx

import "strings"

// TrimFirstPrefix removes the first matching prefix from the string `s` found in the list `prefixes`.
// If no non-empty prefix is found, it returns the original string unchanged.
func TrimFirstPrefix(s string, prefixes ...string) string {
	for _, prefix := range prefixes {
		if prefix == "" {
			continue // Skip empty prefix to ensure it doesn't falsely match
		}
		if strings.HasPrefix(s, prefix) {
			// Remove the prefix by slicing the string from the end of the prefix
			return s[len(prefix):]
		}
	}
	return s
}
