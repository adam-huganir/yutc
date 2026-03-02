package templates

import (
	"fmt"

	"github.com/adam-huganir/yutc/pkg/lexer"
	"github.com/adam-huganir/yutc/pkg/loader"
)

// LoadTemplateInputs loads all Input entries into memory.
func LoadTemplateInputs(tis []*Input) error {
	for _, ti := range tis {
		if err := ti.Load(); err != nil {
			return err
		}
	}
	return nil
}

// ParseTemplateArgs parses raw string arguments and returns [][]*Input per input string.
func ParseTemplateArgs(fs []string, isCommon bool) ([][]*Input, error) {
	result := make([][]*Input, len(fs))
	for i, s := range fs {
		ti, err := ParseTemplateArg(s, isCommon)
		if err != nil {
			return nil, err
		}
		result[i] = []*Input{ti}
	}
	return result, nil
}

// ParseTemplateArg parses a template file argument string into an Input.
func ParseTemplateArg(arg string, isCommon bool) (*Input, error) {
	return ParseTemplateArgWithTempDir(arg, isCommon, "")
}

// ParseTemplateArgWithTempDir parses a template file argument string into an Input,
// configuring git-backed inputs to use tempDir for checkouts.
func ParseTemplateArgWithTempDir(arg string, isCommon bool, tempDir string) (*Input, error) {
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

	ti := NewInput(entryName, isCommon, entryOpts...)

	if sourceType == loader.SourceKindStdin && ti.Name != "-" {
		panic("a bug yo2")
	}

	if argParsed.Auth != nil {
		ti.Auth = loader.ParseAuthString(argParsed.Auth.Value)
	}

	return ti, nil
}
