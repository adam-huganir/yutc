package input

import (
	"fmt"
	"strconv"

	"github.com/adam-huganir/yutc/pkg/lexer"
	"github.com/adam-huganir/yutc/pkg/loader"
)

// ParsedSourceInput contains common parsed structured-input metadata used by
// both data and template argument resolution.
type ParsedSourceInput struct {
	Arg        *lexer.Arg
	SourceType loader.SourceKind
	EntryName  string
	EntryOpts  []loader.FileEntryOption
	Auth       *loader.AuthInfo
}

// ParseSourceInputWithTempDir parses and resolves shared input details (source
// kind, git metadata, and auth), while leaving domain-specific checks to callers.
func ParseSourceInputWithTempDir(arg, tempDir string) (*ParsedSourceInput, error) {
	parser := lexer.NewParser(arg)
	argParsed, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	if argParsed.Source == nil || argParsed.Source.Value == "" {
		return nil, fmt.Errorf("missing or empty 'src' parameter in argument: %s", arg)
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
		recurseSubmodules, err := parseGitSubmodulesArg(argParsed)
		if err != nil {
			return nil, err
		}
		ref := ""
		path := ""
		if argParsed.Ref != nil {
			ref = argParsed.Ref.Value
		}
		if argParsed.Path != nil {
			path = argParsed.Path.Value
		}
		entryOpts = append(entryOpts, loader.WithGitSource(argParsed.Source.Value, ref, path, tempDir, recurseSubmodules))
		entryName = loader.NormalizeGitSourceValue(argParsed.Source.Value)
	}

	var auth *loader.AuthInfo
	if argParsed.Auth != nil {
		parsedAuth := loader.ParseAuthString(argParsed.Auth.Value)
		auth = &parsedAuth
	}

	return &ParsedSourceInput{
		Arg:        argParsed,
		SourceType: sourceType,
		EntryName:  entryName,
		EntryOpts:  entryOpts,
		Auth:       auth,
	}, nil
}

func parseGitSubmodulesArg(argParsed *lexer.Arg) (bool, error) {
	if argParsed == nil || argParsed.Type == nil {
		return false, nil
	}
	if argParsed.Type.Value != string(loader.SourceKindGit) {
		return false, nil
	}
	if argParsed.Type.Args == nil {
		return false, nil
	}

	for argName, argValue := range argParsed.Type.Args {
		if argName != "submodules" {
			return false, fmt.Errorf("invalid argument %q for type=git(): only 'submodules' is allowed", argName)
		}
		recurse, err := strconv.ParseBool(argValue)
		if err != nil {
			return false, fmt.Errorf("invalid value for 'submodules' argument: must be 'true' or 'false'")
		}
		return recurse, nil
	}

	return false, nil
}
