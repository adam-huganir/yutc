package data

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
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

func LoadFileArgsLike(fas []FileArgLike) (err error) {
	for _, fa := range fas {
		f := fa.AsFileArg()
		if f == nil {
			continue
		}
		err = f.Load()
		if err != nil {
			return err
		}
	}
	return nil
}

// ParseFileArgs parses raw string arguments and populates returns []*FileArg.
func ParseFileArgs(fs []string, kind FileKind, _ *zerolog.Logger) ([][]*FileArg, error) {
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

func ParseFileArgsLike(fs []string, kind FileKind, logger *zerolog.Logger) ([][]FileArgLike, error) {
	_ = logger
	fal := make([][]FileArgLike, len(fs))
	for i, stringFileArg := range fs {
		fileArgs, err := ParseFileArgLike(stringFileArg, kind)
		if err != nil {
			return nil, err
		}
		fal[i] = fileArgs
	}
	return fal, nil
}

// ParseFileArg parses a file argument which can be in two formats:
// 1. Simple path: "./my_file.yaml"
// 2. With structure: "path=.Secrets,src=./my_secrets.yaml"
func ParseFileArg(arg string, kind FileKind) (fileArg []*FileArg, err error) {
	fal, err := ParseFileArgLike(arg, kind)
	if err != nil {
		return nil, err
	}
	fileArg = make([]*FileArg, 0, len(fal))
	for _, fa := range fal {
		f := fa.AsFileArg()
		if f == nil {
			continue
		}
		fileArg = append(fileArg, f)
	}
	return fileArg, nil
}

func ParseFileArgLike(arg string, kind FileKind) (fileArg []FileArgLike, err error) {
	// pre: arg is either a file/url/-, or keyed version with src=. kind indicates which arg rules apply.
	parser := lexer.NewParser(arg)

	var argParsed *lexer.Arg
	argParsed, err = parser.Parse()
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

	content := NewFileContent()
	var out FileArgLike
	switch kind {
	case FileKindSchema:
		out = NewSchemaFileArg(argParsed.Source.Value, sourceType, content)
	case FileKindTemplate:
		out = NewTemplateFileArg(argParsed.Source.Value, sourceType, content)
	case FileKindCommonTemplate:
		f := NewTemplateFileArg(argParsed.Source.Value, sourceType, content)
		f.Kind = FileKindCommonTemplate
		out = f
	default:
		out = NewDataFileArg(argParsed.Source.Value, sourceType, content)
	}

	f := out.AsFileArg()
	if f == nil {
		return nil, fmt.Errorf("internal error: nil file arg")
	}
	if sourceType == "stdin" && f.Name != "-" {
		panic("a bug yo2")
	}
	if sourceType == "file" {
		f.Name = NormalizeFilepath(f.Name)
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
				f.DisableSchemaDefaults = !applyDefaults
			}
		}
	}
	if argParsed.Auth != nil {
		if strings.Contains(argParsed.Auth.Value, ":") {
			f.BasicAuth = argParsed.Auth.Value
		} else {
			f.BearerToken = argParsed.Auth.Value
		}
	}

	return []FileArgLike{out}, nil
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
