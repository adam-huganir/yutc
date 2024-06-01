package internal

import "fmt"

var yutcVersion = "0.0.6"

func PrintVersion() {
	fmt.Println(GetVersion())
}

func GetVersion() string {
	return yutcVersion
}
