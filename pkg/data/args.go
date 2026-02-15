package data

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/adam-huganir/yutc/pkg/lexer"
	"github.com/adam-huganir/yutc/pkg/loader"
	"github.com/theory/jsonpath"
)

// ParseFileStringSource re-exported from pkg/loader.
var ParseFileStringSource = loader.ParseFileStringSource

// LoadDataInputs loads all DataInput entries into memory.
func LoadDataInputs(dis []*DataInput) error {
	for _, di := range dis {
		if err := di.Load(); err != nil {
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
