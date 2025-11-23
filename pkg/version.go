package yutc

import "fmt"

var version = "v0.4.0"

// PrintVersion prints the current version to stdout.
func PrintVersion() {
	fmt.Println("yutc " + version)
}

// GetVersion returns the current version string.
func GetVersion() string {
	return version
}
