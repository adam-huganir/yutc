package internal

import "fmt"

var yutcVersion = "0.0.4"

func PrintVersion() {
	fmt.Println(yutcVersion)
}

func GetVersion() string {
	return yutcVersion
}
