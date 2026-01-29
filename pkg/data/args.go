package data

import (
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

// ParseFileArg parses a file argument which can be in two formats:
// 1. Simple path: "./my_file.yaml"
// 2. With structure: "path=.Secrets,src=./my_secrets.yaml"
func ParseFileArg(arg string, kind FileKind) (fileArg []*FileArg, err error) {
	// pre: arg is either a file/url/, or keyed version with src=. kind is either "data" or "schema"
	// post:
	//   - empty file content
	//   - source set to file/url/stdin "enum"
	//   - path set to "-" if stdin, otherwise file path or url based on input
	//   - kind set to "data", or "schema" based on the input keys
	//   - jsonpath set to $ if no jsonpath is provided
	//   - bearerToken and basicAuth set to empty if not structured
	fileArg = []*FileArg{{Kind: kind, JSONPath: jsonpath.MustParse("$"), Content: NewFileContent()}}

	fileArg0 := fileArg[0]
	// TODO: the below won't actually work as pointed out by copilot since fileArg0 does not have name set yet
	isContainer, err := fileArg0.IsContainer()
	if err != nil {
		return nil, err
	} else if isContainer {
		err = fileArg0.CollectContainerChildren()
		if err != nil {
			return nil, err
		}
	}

	parser := lexer.NewParser(arg)

	var argParsed *lexer.Arg
	argParsed, err = parser.Parse()
	if err != nil {
		return nil, err
	}
	if argParsed.Source == nil || argParsed.Source.Value == "" {
		return nil, fmt.Errorf("missing or empty 'src' parameter in argument: %s", arg)
	}

	if argParsed.Source == nil {
		return nil, fmt.Errorf("missing 'src' parameter in argument: %s", arg)
	}
	if kind == FileKindTemplate && argParsed.JSONPath != nil {
		return nil, fmt.Errorf("key parameter is not supported for template arguments: %s", arg)
	} else if argParsed.JSONPath != nil {
		if argParsed.JSONPath.Value[0] != '$' {
			argParsed.JSONPath.Value = "$" + argParsed.JSONPath.Value
		}
		fileArg0.JSONPath, err = jsonpath.Parse(argParsed.JSONPath.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid jsonpath: %s", argParsed.JSONPath)
		}
	}

	fileArg0.Name = argParsed.Source.Value
	if argParsed.Type != nil {
		fk := FileKind(argParsed.Type.Value)
		fileArg0.Kind = fk
	}
	if argParsed.Auth != nil {
		// is this a necessary and sufficient check? tbd
		if strings.Contains(argParsed.Auth.Value, ":") {
			fileArg0.BasicAuth = argParsed.Auth.Value
		} else {
			fileArg0.BearerToken = argParsed.Auth.Value
		}
	}

	if fileArg0.Name == "" {
		return nil, fmt.Errorf("missing 'src' parameter in data argument: %s", arg)
	}

	sourceType, err := ParseFileStringSource(fileArg0.Name)
	if err != nil {
		return nil, err
	}
	fileArg0.Source = sourceType

	if sourceType == "stdin" && fileArg0.Name != "-" {
		panic("a bug yo2")
	}
	if sourceType == "file" {
		fileArg0.Name = NormalizeFilepath(fileArg0.Name)
	}
	return fileArg, nil
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
