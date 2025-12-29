package data

import (
	"encoding/csv"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/adam-huganir/yutc/pkg/lexer"
	"github.com/rs/zerolog"
	"github.com/theory/jsonpath"
)

// LoadFileArgs loads the file args into memory
func LoadFileArgs(fas []*FileArg) (err error) {
	for _, fa := range fas {
		err = fa.Load()
		if err != nil {
			return err
		}
	}
	return nil
}

// ParseFileArgs parses raw string arguments and populates returns []*FileArg.
func ParseFileArgs(fs []string, kind string, logger *zerolog.Logger) ([]*FileArg, error) {
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
func ParseFileArg(arg, kind string) (fileArg *FileArg, err error) {
	// pre: arg is either a file/url/, or keyed version with src=. kind is either "data" or "schema"
	// post:
	//   - empty file content
	//   - source set to file/url/stdin "enum"
	//   - path set to "-" if stdin, otherwise file path or url based on input
	//   - kind set to "data", or "schema" based on the input keys
	//   - jsonpath set to $ if no jsonpath is provided
	//   - bearerToken and basicAuth set to empty if not structured
	fileArg = &FileArg{Kind: kind, JSONPath: jsonpath.MustParse("$"), Content: NewFileContent()}

	parser := lexer.NewParser(arg)

	var argParsed *lexer.Arg
	argParsed, err = parser.Parse()
	if err != nil {
		return nil, err
	}
	if argParsed.Source == nil {
		return nil, fmt.Errorf("missing 'src' parameter in argument: %s", arg)
	}
	fmt.Printf("Parsed arg: %v", argParsed)

	// If either key=, src=, type=, or auth= is present, we expect the structured format. if an equals is in there
	// otherwise we just take that as the filename
	// Use CSV reader to properly parse comma-separated key=value pairs
	for key, value := range argParsed.Map() {
		switch key {
		case "jsonpath":
			if kind == "template" && value != nil {
				return nil, fmt.Errorf("key parameter is not supported for template arguments: %s", arg)
			}
			if value != nil {
				if value.Value[0] != '$' {
					value.Value = "$" + value.Value
				}
				fileArg.JSONPath, err = jsonpath.Parse(value.Value)
				if err != nil {
					return nil, fmt.Errorf("invalid jsonpath: %s", value)
				}
			}
		case "src":
			if value == nil {
				return nil, fmt.Errorf("missing 'src' parameter in argument: %s", arg)
			}
			fileArg.Path = value.Value
		case "type":
			if value != nil {
				fileArg.Kind = value.Value
			}
		case "auth":
			if value != nil {
				// is this a necessary and sufficient check? tbd
				if strings.Contains(value.Value, ":") {
					fileArg.BasicAuth = value.Value
				} else {
					fileArg.BearerToken = value.Value
				}
			}
		default:
			return nil, fmt.Errorf("invalid data argument format with unknown parameter %s: %s", key, arg)
		}
	}
	if fileArg.Path == "" {
		return nil, fmt.Errorf("missing 'src' parameter in data argument: %s", arg)
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
