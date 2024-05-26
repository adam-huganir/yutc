package internal

func must(result any, err error) any {
	if err != nil {
		panic(err)
	}
	return result
}

func CmpStringLength(a, b string) int {
	if len(a) < len(b) {
		return -1
	}
	return 1
}
