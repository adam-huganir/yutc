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

	sourceType, err := loader.ParseFileStringSource(argParsed.Source.Value)
	if err != nil {
		return nil, err
	}

	ti := NewInput(argParsed.Source.Value, isCommon, loader.WithSource(sourceType))

	if sourceType == loader.SourceKindStdin && ti.Name != "-" {
		panic("a bug yo2")
	}

	if argParsed.Auth != nil {
		ti.Auth = loader.ParseAuthString(argParsed.Auth.Value)
	}

	return ti, nil
}
