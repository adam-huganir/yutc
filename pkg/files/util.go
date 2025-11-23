package files

// CmpStringLength compares two strings by length, returning -1 if a is shorter, otherwise 1.
func CmpStringLength(a, b string) int {
	if len(a) < len(b) {
		return -1
	}
	return 1
}
