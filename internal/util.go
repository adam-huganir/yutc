package internal

func must[R any](result R, err error) R {
	if err != nil {
		panic(err)
	}
	return result
}

func CmpTemplatePathLength(a, b *YutcTemplate) int {
	return CmpStringLength((*a).Path(), (*b).Path())
}

func CmpStringLength(a, b string) int {
	if len(a) < len(b) {
		return -1
	}
	return 1
}
