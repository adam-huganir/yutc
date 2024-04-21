package internal

import "fmt"

var yutcVersion = "0.0.5"

func PrintVersion() {
	fmt.Println(GetVersion())
}

func GetVersion() string {
	return yutcVersion
}
