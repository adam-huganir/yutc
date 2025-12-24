package data

import (
	"encoding/csv"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/theory/jsonpath"
)

// ParseFileArgs parses raw string arguments and populates returns []*FileArg.
func ParseFileArgs(fs []string, kind string) ([]*FileArg, error) {
	fas := make([]*FileArg, len(fs))
	for i, stringFileArg := range fs {
		fileArg, err := ParseFileArg(stringFileArg, kind)
		if err != nil {
			return nil, err
		}
		fas[i] = fileArg
	}
	return fas, nil
}

// ParseFileArg parses a file argument which can be in two formats:
// 1. Simple path: "./my_file.yaml"
// 2. With structure: "path=.Secrets,src=./my_secrets.yaml"
func ParseFileArg(arg, kind string) (*FileArg, error) {
	fileArg := &FileArg{Kind: kind, JSONPath: jsonpath.MustParse("$"), Content: &FileContent{}}

	// Check if the argument contains the structured format
	isStructured := false
	for _, key := range []string{
		"jsonpath", "src", "type", "auth",
	} {
		isStructured = strings.Contains(arg, key+"=")
		if isStructured {
			break
		}
	}

	// If either key=, src=, type=, or auth= is present, we expect the structured format. if an equals is in there
	// otherwise we just take that as the filename
	if isStructured {
		// Use CSV reader to properly parse comma-separated key=value pairs
		data, err := mapFromKeyValueOption(arg)
		if err != nil {
			return nil, err
		}

		for key, value := range data {
			switch key {
			case "jsonpath":
				if kind == "template" {
					return nil, fmt.Errorf("key parameter is not supported for template arguments: %s", arg)
				}
				if value[0] != '$' {
					value = "$" + value
				}
				fileArg.JSONPath, err = jsonpath.Parse(value)
				if err != nil {
					return nil, fmt.Errorf("invalid jsonpath: %s", value)
				}
			case "src":
				fileArg.Path = value
			case "type":
				fileArg.Kind = value
			case "auth":
				// is this a necessary and sufficient check? tbd
				if strings.Contains(value, ":") {
					fileArg.BasicAuth = value
				} else {
					fileArg.BearerToken = value
				}
			default:
				return nil, fmt.Errorf("invalid data argument format with unknown parameter %s: %s", key, arg)
			}
		}
		if fileArg.Path == "" {
			return nil, fmt.Errorf("missing 'src' parameter in data argument: %s", arg)
		}

	} else {
		// just a simple path
		fileArg.Path = arg
	}

	sourceType, err := ParseFileStringSource(fileArg.Path)
	if err != nil {
		return nil, err
	}
	fileArg.Source = sourceType

	if sourceType == "stdin" && fileArg.Path != "-" {
		panic("a bug yo2")
	}
	if sourceType == "file" {
		fileArg.Path = NormalizeFilepath(fileArg.Path)
	}

	return fileArg, nil
}

func mapFromKeyValueOption(arg string) (map[string]string, error) {
	reader := csv.NewReader(strings.NewReader(arg))
	reader.TrimLeadingSpace = true

	records, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to parse data argument: %w", err)
	}

	data := make(map[string]string)

	for _, part := range records {
		if !strings.Contains(part, "=") {
			return nil, fmt.Errorf("invalid data argument format, no argument provided in %s: %s", part, arg)
		}
		part = strings.TrimSpace(part)
		prefix := part[:strings.Index(part, "=")]  //nolint: gocritic // we already know "=" exists
		value := part[strings.Index(part, "=")+1:] //nolint: gocritic // we already know "=" exists
		data[prefix] = value
	}
	return data, nil
}

// ParseFileStringSource determines the source of a file string flag based on format and returns the source
// as a string, or an error if the source is not supported. Currently, supports "file", "url", and "stdin" (as `-`).
func ParseFileStringSource(v string) (string, error) {
	if !strings.Contains(v, "://") {
		if v == "-" {
			return "stdin", nil
		}
		_, err := filepath.Abs(v)
		if err != nil {
			return "", err
		}
		return "file", nil
	}
	if v == "-" {
		return "stdin", nil
	}
	allowedURLPrefixes := []string{"http://", "https://"}
	for _, prefix := range allowedURLPrefixes {
		if strings.HasPrefix(v, prefix) {
			return "url", nil
		}
	}
	return "", errors.New("unsupported scheme/source for input: " + v)
}
