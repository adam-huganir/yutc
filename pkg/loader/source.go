package loader

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

var knownGitHosts = map[string]bool{
	"github.com":    true,
	"gitlab.com":    true,
	"bitbucket.org": true,
}

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

// LooksLikeGitSource reports whether a source value should be treated as a git repository input.
func LooksLikeGitSource(value string) bool {
	v := strings.TrimSpace(value)
	if v == "" {
		return false
	}

	if strings.HasPrefix(v, "git@") || strings.HasPrefix(v, "ssh://") || strings.HasSuffix(strings.ToLower(v), ".git") {
		return true
	}

	if host, parts, ok := parseHostAndPath(v); ok {
		if knownGitHosts[strings.ToLower(host)] && len(parts) >= 2 {
			return true
		}
	}

	return false
}

// NormalizeGitSourceValue normalizes known-host git sources by prepending https:// when no scheme is provided.
func NormalizeGitSourceValue(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return v
	}
	if strings.Contains(v, "://") || strings.HasPrefix(v, "git@") || strings.HasPrefix(v, "ssh://") {
		return v
	}
	if filepath.VolumeName(v) != "" || strings.HasPrefix(v, ".") || strings.HasPrefix(v, "/") || strings.Contains(v, "\\") {
		return v
	}
	if host, parts, ok := parseHostAndPath(v); ok {
		if knownGitHosts[strings.ToLower(host)] && len(parts) >= 2 {
			return "https://" + v
		}
	}
	return v
}

func parseHostAndPath(value string) (host string, parts []string, ok bool) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil, false
	}
	if strings.Contains(v, " ") || strings.Contains(v, "\t") {
		return "", nil, false
	}

	parsed, err := url.Parse(v)
	if err == nil && parsed.Host != "" {
		host = strings.Split(parsed.Host, ":")[0]
		trimmed := strings.Trim(parsed.Path, "/")
		if trimmed != "" {
			parts = strings.Split(trimmed, "/")
		}
		return host, parts, true
	}

	withScheme, err := url.Parse("https://" + v)
	if err != nil || withScheme.Host == "" {
		return "", nil, false
	}
	host = strings.Split(withScheme.Host, ":")[0]
	trimmed := strings.Trim(withScheme.Path, "/")
	if trimmed != "" {
		parts = strings.Split(trimmed, "/")
	}
	return host, parts, true
}
