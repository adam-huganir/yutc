package main

import (
	"os"
)

func checkIfDir(path string) (*bool, error) {
	var b bool
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		YutcLog.Error().Msg(err.Error())
	}
	if stat.IsDir() {
		b = true
	} else {
		b = false
	}
	return &b, nil
}
