package loader

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// ParseFileStringSource determines the source of a file string flag based on format and returns the source
// as a SourceKind, or an error if the source is not supported. Currently, supports "file", "url", and "stdin" (as `-`).
func ParseFileStringSource(v string) (SourceKind, error) {
	if v == "-" {
		return SourceKindStdin, nil
	}
	if strings.Contains(v, "://") {
		allowedURLPrefixes := []string{"http://", "https://"}
		for _, prefix := range allowedURLPrefixes {
			if strings.HasPrefix(v, prefix) {
				return SourceKindURL, nil
			}
		}
		return "", errors.New("unsupported scheme/source for input: " + v)
	}

	exists, err := Exists(v)
	if err != nil {
		if looksLikeURL(v) {
			return SourceKindURL, nil
		}
		return "", err
	}
	if exists {
		return SourceKindFile, nil
	}
	if looksLikeURL(v) {
		return SourceKindURL, nil
	}
	_, err = filepath.Abs(v)
	if err != nil {
		return "", err
	}
	return SourceKindFile, nil
}

func looksLikeURL(value string) bool {
	if value == "" {
		return false
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	if strings.Contains(value, "=") {
		return false
	}
	if strings.HasPrefix(value, ".") || strings.HasPrefix(value, "/") {
		return false
	}
	if filepath.VolumeName(value) != "" {
		return false
	}
	if strings.Contains(value, "\\") {
		return false
	}
	parsed, err := url.Parse("https://" + value)
	if err != nil || parsed.Host == "" {
		return false
	}
	host := strings.Split(parsed.Host, ":")[0]
	if host == "localhost" {
		return true
	}
	return strings.Count(host, ".") >= 1
}

func normalizeURLString(value string) string {
	if value == "" {
		return value
	}
	if strings.Contains(value, "://") {
		return value
	}
	return "https://" + value
}

// ParseSourceKind converts an explicit source kind string into a SourceKind.
func ParseSourceKind(value string) (SourceKind, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(SourceKindFile):
		return SourceKindFile, nil
	case string(SourceKindURL):
		return SourceKindURL, nil
	case string(SourceKindStdin):
		return SourceKindStdin, nil
	case string(SourceKindStdout):
		return SourceKindStdout, nil
	case string(SourceKindGit):
		return SourceKindGit, nil
	case "":
		return "", fmt.Errorf("source kind is empty")
	default:
		return "", fmt.Errorf("invalid source kind: %s", value)
	}
}
