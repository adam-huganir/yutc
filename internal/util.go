package internal

func must(result any, err error) any {
	if err != nil {
		panic(err)
	}
	return result
}
