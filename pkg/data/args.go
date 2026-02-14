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

// ParseFileArgs parses raw string arguments and returns []*FileArg per input string.
func ParseFileArgs(fs []string, kind FileKind) ([][]*FileArg, error) {
	fas := make([][]*FileArg, len(fs))
	for i, stringFileArg := range fs {
		fileArgs, err := ParseFileArg(stringFileArg, kind)
		if err != nil {
			return nil, err
		}
		fas[i] = fileArgs
	}
	return fas, nil
}

// ParseFileArg parses a file argument which can be in two formats:
// 1. Simple path: "./my_file.yaml"
// 2. With structure: "path=.Secrets,src=./my_secrets.yaml"
func ParseFileArg(arg string, kind FileKind) ([]*FileArg, error) {
	parser := lexer.NewParser(arg)

	argParsed, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	if argParsed.Source == nil || argParsed.Source.Value == "" {
		return nil, fmt.Errorf("missing or empty 'src' parameter in argument: %s", arg)
	}

	if kind == FileKindTemplate || kind == FileKindCommonTemplate {
		if argParsed.JSONPath != nil {
			return nil, fmt.Errorf("key parameter is not supported for template arguments: %s", arg)
		}
	} else if argParsed.JSONPath != nil {
		if argParsed.JSONPath.Value != "" && argParsed.JSONPath.Value[0] != '$' {
			argParsed.JSONPath.Value = "$" + argParsed.JSONPath.Value
		}
	}

	sourceType, err := ParseFileStringSource(argParsed.Source.Value)
	if err != nil {
		return nil, err
	}

	// Build options based on kind
	opts := []FileArgOption{WithKind(kind), WithSource(sourceType)}
	if kind == FileKindData || kind == FileKindSchema || kind == "" {
		opts = append(opts, WithDefaultJSONPath())
	}

	f := NewFileArg(argParsed.Source.Value, opts...)

	if sourceType == SourceKindStdin && f.Name != "-" {
		panic("a bug yo2")
	}

	if kind != FileKindTemplate && kind != FileKindCommonTemplate {
		if argParsed.JSONPath != nil {
			f.JSONPath, err = jsonpath.Parse(argParsed.JSONPath.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid jsonpath: %s", argParsed.JSONPath)
			}
		}
	}

	if argParsed.Type != nil {
		fk := FileKind(argParsed.Type.Value)
		f.Kind = fk
		if fk == FileKindSchema {
			if defaultsValue, ok := argParsed.Type.Args["defaults"]; ok {
				applyDefaults, err := strconv.ParseBool(defaultsValue)
				if err != nil {
					return nil, fmt.Errorf("invalid defaults value %q: %w", defaultsValue, err)
				}
				f.Schema.DisableDefaults = !applyDefaults
			}
		}
	}
	if argParsed.Auth != nil {
		if strings.Contains(argParsed.Auth.Value, ":") {
			f.Auth.BasicAuth = argParsed.Auth.Value
		} else {
			f.Auth.BearerToken = argParsed.Auth.Value
		}
	}

	return []*FileArg{f}, nil
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
