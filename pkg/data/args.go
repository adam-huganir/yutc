package data

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adam-huganir/yutc/pkg/lexer"
	"github.com/theory/jsonpath"
)

// LoadDataInputs loads all DataInput entries into memory.
func LoadDataInputs(dis []*DataInput) error {
	for _, di := range dis {
		if err := di.Load(); err != nil {
			return err
		}
	}
	return nil
}

// LoadTemplateInputs loads all TemplateInput entries into memory.
func LoadTemplateInputs(tis []*TemplateInput) error {
	for _, ti := range tis {
		if err := ti.Load(); err != nil {
			return err
		}
	}
	return nil
}

// ParseDataArgs parses raw string arguments and returns [][]*DataInput per input string.
func ParseDataArgs(fs []string) ([][]*DataInput, error) {
	result := make([][]*DataInput, len(fs))
	for i, s := range fs {
		dis, err := ParseDataArg(s)
		if err != nil {
			return nil, err
		}
		result[i] = dis
	}
	return result, nil
}

// ParseTemplateArgs parses raw string arguments and returns [][]*TemplateInput per input string.
func ParseTemplateArgs(fs []string, isCommon bool) ([][]*TemplateInput, error) {
	result := make([][]*TemplateInput, len(fs))
	for i, s := range fs {
		ti, err := ParseTemplateArg(s, isCommon)
		if err != nil {
			return nil, err
		}
		result[i] = []*TemplateInput{ti}
	}
	return result, nil
}

// ParseDataArg parses a data file argument string into one or more DataInput entries.
// Supports simple paths ("./my_file.yaml") and structured args ("jsonpath=.Secrets,src=./my_secrets.yaml").
func ParseDataArg(arg string) ([]*DataInput, error) {
	parser := lexer.NewParser(arg)

	argParsed, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	if argParsed.Source == nil || argParsed.Source.Value == "" {
		return nil, fmt.Errorf("missing or empty 'src' parameter in argument: %s", arg)
	}

	if argParsed.JSONPath != nil {
		if argParsed.JSONPath.Value != "" && argParsed.JSONPath.Value[0] != '$' {
			argParsed.JSONPath.Value = "$" + argParsed.JSONPath.Value
		}
	}

	sourceType, err := ParseFileStringSource(argParsed.Source.Value)
	if err != nil {
		return nil, err
	}

	entryOpts := []FileEntryOption{WithSource(sourceType)}
	dataOpts := []DataInputOption{WithDefaultJSONPath()}

	di := NewDataInput(argParsed.Source.Value, entryOpts, dataOpts...)

	if sourceType == SourceKindStdin && di.Name != "-" {
		panic("a bug yo2")
	}

	if argParsed.JSONPath != nil {
		di.JSONPath, err = jsonpath.Parse(argParsed.JSONPath.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid jsonpath: %s", argParsed.JSONPath)
		}
	}

	if argParsed.Type != nil {
		if argParsed.Type.Value == "schema" {
			di.IsSchema = true
			if defaultsValue, ok := argParsed.Type.Args["defaults"]; ok {
				applyDefaults, err := strconv.ParseBool(defaultsValue)
				if err != nil {
					return nil, fmt.Errorf("invalid defaults value %q: %w", defaultsValue, err)
				}
				di.Schema.DisableDefaults = !applyDefaults
			}
		}
	}

	if argParsed.Auth != nil {
		if strings.Contains(argParsed.Auth.Value, ":") {
			di.Auth.BasicAuth = argParsed.Auth.Value
		} else {
			di.Auth.BearerToken = argParsed.Auth.Value
		}
	}

	return []*DataInput{di}, nil
}

// ParseTemplateArg parses a template file argument string into a TemplateInput.
func ParseTemplateArg(arg string, isCommon bool) (*TemplateInput, error) {
	parser := lexer.NewParser(arg)

	argParsed, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	if argParsed.Source == nil || argParsed.Source.Value == "" {
		return nil, fmt.Errorf("missing or empty 'src' parameter in argument: %s", arg)
	}

	if argParsed.JSONPath != nil {
		return nil, fmt.Errorf("key parameter is not supported for template arguments: %s", arg)
	}

	sourceType, err := ParseFileStringSource(argParsed.Source.Value)
	if err != nil {
		return nil, err
	}

	ti := NewTemplateInput(argParsed.Source.Value, isCommon, WithSource(sourceType))

	if sourceType == SourceKindStdin && ti.Name != "-" {
		panic("a bug yo2")
	}

	if argParsed.Auth != nil {
		if strings.Contains(argParsed.Auth.Value, ":") {
			ti.Auth.BasicAuth = argParsed.Auth.Value
		} else {
			ti.Auth.BearerToken = argParsed.Auth.Value
		}
	}

	return ti, nil
}

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
