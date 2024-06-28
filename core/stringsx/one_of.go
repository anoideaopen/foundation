package stringsx

// OneOf checks if a given string s is present within a slice of strings ss.
func OneOf(s string, ss ...string) bool {
	set := make(map[string]struct{})
	for _, v := range ss {
		set[v] = struct{}{}
	}

	_, exists := set[s]
	return exists
}
