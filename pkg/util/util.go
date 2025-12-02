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

// Dedent removes a common leading prefix from each line in the input string s.
// If a prefix is specified, it uses that prefix; otherwise, it determines the
// common leading whitespace from the first non-empty line.
// It returns an error if any line (except an optional first empty line for formatting reasons) does not start
// with the specified or determined prefix.
// Special Notes:
// - If first line is a single newline, it is discarded to allow for prettier formatting in code.
// - If the last
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
		if i != 0 && strings.TrimSpace(line) != "" && !strings.HasPrefix(line, prefixActual) {
			return "", &types.DedentError{Line: line, Prefix: prefixActual}
		} else if i == 0 && line == "" {
			// Allow first line to be empty (exactly "\n") and discard it to allow for prettier formatting
			//in code
			continue
		} else if strings.TrimSpace(line) == "" {
			// allow lines that are only whitespace to become empty lines, even if they don't have the prefix
			out = append(out, "")
		} else {
			out = append(out, strings.TrimPrefix(line, prefixActual))
		}
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
