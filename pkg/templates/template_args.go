package templates

import (
	"fmt"

	inputpkg "github.com/adam-huganir/yutc/pkg/input"
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
	parsed, err := inputpkg.ParseSourceInputWithTempDir(arg, tempDir)
	if err != nil {
		return nil, err
	}
	argParsed := parsed.Arg

	if argParsed.JSONPath != nil {
		return nil, fmt.Errorf("key parameter is not supported for template arguments: %s", arg)
	}

	ti := NewInput(parsed.EntryName, isCommon, parsed.EntryOpts...)

	if parsed.SourceType.String() == "stdin" && ti.Name != "-" {
		panic("a bug yo2")
	}

	if parsed.Auth != nil {
		ti.Auth = *parsed.Auth
	}

	return ti, nil
}
