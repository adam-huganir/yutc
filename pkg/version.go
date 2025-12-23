package yutc

import "fmt"

var version = "0.5.0"

// PrintVersion prints the current version to stdout.
func PrintVersion() {
	fmt.Println(version)
}

// GetVersion returns the current version string.
func GetVersion() string {
	return version
}
