package loader

import (
	"errors"
	"path/filepath"
	"strings"
)

// ParseFileStringSource determines the source of a file string flag based on format and returns the source
// as a SourceKind, or an error if the source is not supported. Currently, supports "file", "url", and "stdin" (as `-`).
func ParseFileStringSource(v string) (SourceKind, error) {
	if !strings.Contains(v, "://") {
		if v == "-" {
			return SourceKindStdin, nil
		}
		_, err := filepath.Abs(v)
		if err != nil {
			return "", err
		}
		return SourceKindFile, nil
	}
	if v == "-" {
		return SourceKindStdin, nil
	}
	allowedURLPrefixes := []string{"http://", "https://"}
	for _, prefix := range allowedURLPrefixes {
		if strings.HasPrefix(v, prefix) {
			return SourceKindURL, nil
		}
	}
	return "", errors.New("unsupported scheme/source for input: " + v)
}
