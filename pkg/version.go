package yutc

import "fmt"

var version = "0.7.0"

// PrintVersion prints the current version to stdout.
func PrintVersion() {
	fmt.Println(GetVersion())
}

// GetVersion returns the current version string.
func GetVersion() string {
	return version
}
