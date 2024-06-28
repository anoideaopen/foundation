package stringsx

import "strings"

// HasPrefix checks if the given string s has any prefix from the provided list of prefixes.
// It returns true if any prefix matches the start of s, otherwise returns false.
func HasPrefix(s string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
