package yutc

import "fmt"

var yutcVersion = "0.3.3"

func PrintVersion() {
	fmt.Println(GetVersion())
}

func GetVersion() string {
	return yutcVersion
}
