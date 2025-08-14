package internal

import "fmt"

var yutcVersion = "0.2.0"

func PrintVersion() {
	fmt.Println(GetVersion())
}

func GetVersion() string {
	return yutcVersion
}
