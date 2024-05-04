package internal

func Must(result any, err error) any {
	if err != nil {
		panic(err)
	}
	return result
}
