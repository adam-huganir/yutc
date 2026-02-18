package data

import (
	"fmt"
	"strconv"

	"github.com/adam-huganir/yutc/pkg/lexer"
	"github.com/adam-huganir/yutc/pkg/loader"
	"github.com/theory/jsonpath"
)

// LoadDataInputs loads all Input entries into memory.
func LoadDataInputs(dis []*Input) error {
	for _, di := range dis {
		if err := di.Load(); err != nil {
			return err
		}
	}
	return nil
}

// ParseDataArgs parses raw string arguments and returns [][]*Input per input string.
func ParseDataArgs(fs []string) ([][]*Input, error) {
	result := make([][]*Input, len(fs))
	for i, s := range fs {
		dis, err := ParseDataArg(s)
		if err != nil {
			return nil, err
		}
		result[i] = dis
	}
	return result, nil
}

// ParseDataArg parses a data file argument string into one or more Input entries.
// Supports simple paths ("./my_file.yaml") and structured args ("jsonpath=.Secrets,src=./my_secrets.yaml").
func ParseDataArg(arg string) ([]*Input, error) {
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

	sourceType, err := loader.ParseFileStringSource(argParsed.Source.Value)
	if err != nil {
		return nil, err
	}

	entryOpts := []loader.FileEntryOption{loader.WithSource(sourceType)}
	dataOpts := []InputOption{WithDefaultJSONPath()}

	di := NewInput(argParsed.Source.Value, entryOpts, dataOpts...)

	if sourceType == loader.SourceKindStdin && di.Name != "-" {
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
		di.Auth = loader.ParseAuthString(argParsed.Auth.Value)
	}

	return []*Input{di}, nil
}
