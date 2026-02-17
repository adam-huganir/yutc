package loader

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
)

// tailMarker prints the file name as inspired by coreutils tail
func tailMarker(path string) string {
	return strings.Join([]string{"==>", path, "<=="}, " ")
}

func TailMergeFiles(paths []string) (out string, err error) {
	if len(paths) == 0 {
		return "", nil
	}
	var copied []string
	for _, path := range paths {
		if !slices.Contains(copied, path) {
			copied = append(copied, path)
		}
	}
	sort.Strings(copied)
	for _, path := range copied {
		exists, err := Exists(path)
		if err != nil {
			return "", err
		} else if !exists {
			return "", fmt.Errorf("file %s does not exist", path)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		asString := string(content)
		if len(copied) == 1 {
			return asString, nil
		}
		out += tailMarker(path) + "\n"
		out += asString + "\n"
	}
	return strings.TrimSuffix(out, "\n"), nil // get rid of last newline
}
