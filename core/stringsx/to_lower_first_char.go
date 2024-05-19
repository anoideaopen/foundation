package stringsx

import (
	"strings"
	"unicode/utf8"
)

// LowerFirstChar takes a string and returns a new string with the first character converted to lowercase.
func LowerFirstChar(s string) string {
	if s == "" {
		return ""
	}

	// Decode the first rune in the string.
	firstRune, size := utf8.DecodeRuneInString(s)

	// Convert the first rune to lowercase.
	lowerFirstRune := strings.ToLower(string(firstRune))

	// Return the new string with the first rune in lowercase and the rest of the string unchanged.
	return lowerFirstRune + s[size:]
}
