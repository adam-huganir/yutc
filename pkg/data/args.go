package data

import (
	"fmt"
	"strconv"

	inputpkg "github.com/adam-huganir/yutc/pkg/input"
	"github.com/adam-huganir/yutc/pkg/lexer"
	"github.com/theory/jsonpath"
)

func applyDataKindOptions(di *Input, kind *lexer.KindField) error {
	if kind == nil {
		return nil
	}

	if kind.Value != "schema" {
		return fmt.Errorf("invalid kind %q: only 'schema' is supported", kind.Value)
	}

	di.IsSchema = true

	for argName, argValue := range kind.Args {
		if argName != "defaults" {
			return fmt.Errorf("invalid argument %q for kind=schema(): only 'defaults' is allowed", argName)
		}
		applyDefaults, err := strconv.ParseBool(argValue)
		if err != nil {
			return fmt.Errorf("invalid value for 'defaults' argument: must be 'true' or 'false'")
		}
		di.Schema.DisableDefaults = !applyDefaults
	}

	return nil
}

// ParseDataArgWithTempDir parses a data file argument string and configures git inputs to use tempDir for checkouts.
func ParseDataArgWithTempDir(arg, tempDir string) ([]*Input, error) {
	parsed, err := inputpkg.ParseSourceInputWithTempDir(arg, tempDir)
	if err != nil {
		return nil, err
	}
	argParsed := parsed.Arg

	if argParsed.JSONPath != nil {
		if argParsed.JSONPath.Value != "" && argParsed.JSONPath.Value[0] != '$' {
			argParsed.JSONPath.Value = "$" + argParsed.JSONPath.Value
		}
	}
	dataOpts := []InputOption{WithDefaultJSONPath()}

	di := NewInput(parsed.EntryName, parsed.EntryOpts, dataOpts...)

	if parsed.SourceType.String() == "stdin" && di.Name != "-" {
		panic("a bug yo2")
	}

	if argParsed.JSONPath != nil {
		di.JSONPath, err = jsonpath.Parse(argParsed.JSONPath.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid jsonpath: %s", argParsed.JSONPath)
		}
	}

	if err := applyDataKindOptions(di, argParsed.Kind); err != nil {
		return nil, err
	}

	if parsed.Auth != nil {
		di.Auth = *parsed.Auth
	}

	return []*Input{di}, nil
}
