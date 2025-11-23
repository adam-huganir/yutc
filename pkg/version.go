package yutc

import "fmt"

var yutcVersion = "0.4.0"

func PrintVersion() {
	fmt.Println(GetVersion())
}

func GetVersion() string {
	return yutcVersion
}
