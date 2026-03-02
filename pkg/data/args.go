package data

import (
	"fmt"
	"strconv"

	"github.com/adam-huganir/yutc/pkg/lexer"
	"github.com/adam-huganir/yutc/pkg/loader"
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
	return ParseDataArgWithTempDir(arg, "")
}

// ParseDataArgWithTempDir parses a data file argument string and configures git inputs to use tempDir for checkouts.
func ParseDataArgWithTempDir(arg, tempDir string) ([]*Input, error) {
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

	var sourceType loader.SourceKind
	if argParsed.Type != nil && argParsed.Type.Value != "" {
		sourceType, err = loader.ParseSourceKind(argParsed.Type.Value)
		if err != nil {
			return nil, err
		}
	} else {
		sourceType, err = loader.ParseFileStringSource(argParsed.Source.Value)
		if err != nil {
			if loader.LooksLikeGitSource(argParsed.Source.Value) {
				sourceType = loader.SourceKindGit
			} else {
				return nil, err
			}
		}
	}
	if argParsed.Ref != nil || argParsed.Path != nil || loader.LooksLikeGitSource(argParsed.Source.Value) {
		sourceType = loader.SourceKindGit
	}
	if sourceType == loader.SourceKindStdin && argParsed.Source.Value != "-" {
		return nil, fmt.Errorf("stdin source requires src to be '-': %s", arg)
	}

	entryOpts := []loader.FileEntryOption{loader.WithSource(sourceType)}
	entryName := argParsed.Source.Value
	if sourceType == loader.SourceKindGit {
		ref := ""
		path := ""
		if argParsed.Ref != nil {
			ref = argParsed.Ref.Value
		}
		if argParsed.Path != nil {
			path = argParsed.Path.Value
		}
		entryOpts = append(entryOpts, loader.WithGitSource(argParsed.Source.Value, ref, path, tempDir))
		entryName = loader.NormalizeGitSourceValue(argParsed.Source.Value)
	}
	dataOpts := []InputOption{WithDefaultJSONPath()}

	di := NewInput(entryName, entryOpts, dataOpts...)

	if sourceType == loader.SourceKindStdin && di.Name != "-" {
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

	if argParsed.Auth != nil {
		di.Auth = loader.ParseAuthString(argParsed.Auth.Value)
	}

	return []*Input{di}, nil
}
