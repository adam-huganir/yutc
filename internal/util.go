package internal

func CmpStringLength(a, b string) int {
	if len(a) < len(b) {
		return -1
	}
	return 1
}
