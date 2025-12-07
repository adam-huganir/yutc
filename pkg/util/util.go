package util

import (
	"strings"

	"github.com/adam-huganir/yutc/pkg/types"
)

// AnyStringFunc returns true if any element in the slice satisfies the predicate function f.
func AnyStringFunc(s []string, f func(string) bool) bool {
	for i := range s {
		if f(s[i]) {
			return true
		}
	}
	return false
}

// AllStringFunc returns true if all elements in the slice satisfy the predicate function f.
func AllStringFunc(s []string, f func(string) bool) bool {
	for i := range s {
		if !f(s[i]) {
			return false
		}
	}
	return true
}

// Dedent removes a common leading whitespace prefix from each line in the input string.
// If a `prefix` is provided, it will be used as the prefix to remove.
// If no `prefix` is provided, the leading whitespace of the first non-empty line is used as the prefix for all subsequent lines.
// An empty first line is ignored, which allows for more readable multiline strings in code.
// Lines containing only whitespace are treated as empty lines.
// It returns an error if a line (that is not empty) does not have the determined prefix.
func Dedent(s string, prefix ...string) (string, error) {
	var out []string
	var prefixActual string
	prefixSpecified := len(prefix) > 0
	lines := strings.Split(s, "\n")

	if prefixSpecified {
		prefixActual = prefix[0]
	} else {
		// Determine common leading whitespace from the first non-empty line
		for _, line := range lines {
			trimmed := strings.TrimLeft(line, " \t")
			if trimmed != "" {
				prefixActual = line[:len(line)-len(trimmed)]
				break
			}
		}
	}

	for i, line := range lines {
		switch i {
		case 0:
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, prefixActual) {
				return "", &types.DedentError{Line: line, Prefix: prefixActual}
			} else if line == "" {
				// Allow first line to be empty (exactly "\n") and discard it to allow for prettier formatting
				// in code
				continue
			}
		default:
			if strings.TrimSpace(line) == "" {
				// allow lines that are only whitespace to become empty lines, even if they don't have the prefix
				line = prefixActual
			} else if !strings.HasPrefix(line, prefixActual) {
				return "", &types.DedentError{Line: line, Prefix: prefixActual}
			}
		}
		out = append(out, strings.TrimPrefix(line, prefixActual))
	}
	return strings.Join(out, "\n"), nil
}

func MustDedent(s string, prefix ...string) string {
	dedented, err := Dedent(s, prefix...)
	if err != nil {
		panic(err)
	}
	return dedented
}
