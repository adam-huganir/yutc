package config

import (
	"slices"
	"testing"

	"github.com/adam-huganir/yutc/pkg/data"
	"github.com/adam-huganir/yutc/pkg/templates"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// parsedFromArgs builds a ParsedInputs from raw Arguments strings (parse-once helper for tests).
func parsedFromArgs(t *testing.T, args *types.Arguments) *ParsedInputs {
	t.Helper()
	parsed := &ParsedInputs{}
	if len(args.DataFiles) > 0 {
		da, err := data.ParseDataArgs(args.DataFiles)
		if err != nil {
			t.Fatalf("ParseDataArgs: %v", err)
		}
		parsed.DataFiles = slices.Concat(da...)
	}
	if len(args.TemplatePaths) > 0 {
		tp, err := templates.ParseTemplateArgs(args.TemplatePaths, false)
		if err != nil {
			t.Fatalf("ParseTemplateArgs: %v", err)
		}
		parsed.TemplateFiles = slices.Concat(tp...)
	}
	if len(args.CommonTemplateFiles) > 0 {
		ct, err := templates.ParseTemplateArgs(args.CommonTemplateFiles, true)
		if err != nil {
			t.Fatalf("ParseTemplateArgs (common): %v", err)
		}
		parsed.CommonTemplateFiles = slices.Concat(ct...)
	}
	return parsed
}

func TestValidateArguments(t *testing.T) {
	logger := zerolog.Nop()

	args := &types.Arguments{
		DataFiles:           []string{"../../testFiles/data/data1.yaml", "../../testFiles/data/data2.yaml"},
		CommonTemplateFiles: []string{"../../testFiles/common/common1.tmpl"},
		TemplatePaths:       []string{"../../testFiles/templates/template1.tmpl", "../../testFiles/templates/template2.tmpl"},
		Output:              "../../testFiles/outputs",
	}
	err := ValidateArguments(args, parsedFromArgs(t, args), &logger)
	assert.NoError(t, err, "this is a valid set of inputs")

	args = &types.Arguments{
		DataFiles:     []string{"-"},
		TemplatePaths: []string{"../../testFiles/templates/template1.tmpl"},
		Output:        "-",
	}
	err = ValidateArguments(args, parsedFromArgs(t, args), &logger)
	assert.NoError(t, err, "also valid, only 1 stdin and 1 stdout")

	args = &types.Arguments{
		DataFiles:     []string{"-"},
		TemplatePaths: []string{"-", "../../testFiles/templates/template2.tmpl"},
		Output:        ".",
	}
	err = ValidateArguments(args, parsedFromArgs(t, args), &logger)
	assert.Error(t, err, "you can't specify stdin as multiple things")
	assert.IsType(t, &types.ValidationError{}, err, "should be a ValidationError")

	args = &types.Arguments{
		DataFiles:           []string{"-", "../../testFiles/data/data2.yaml"},
		CommonTemplateFiles: []string{"../../testFiles/common/common1.tmpl"},
		TemplatePaths:       []string{"../../testFiles/templates/template2.tmpl"},
		Output:              "out.yaml",
	}
	err = ValidateArguments(args, parsedFromArgs(t, args), &logger)
	assert.NoError(t, err, "this is a valid set of inputs")

	args = &types.Arguments{
		DataFiles:           []string{"-", "../../testFiles/data/data2.yaml"},
		CommonTemplateFiles: []string{"../../testFiles/common/common1.tmpl"},
		TemplatePaths:       []string{"../../testFiles/templates/template2.tmpl"},
		Output:              "../../testFiles/data/data1.yaml",
	}
	err = ValidateArguments(args, parsedFromArgs(t, args), &logger)
	assert.Error(t, err, "file exists and overwrite is not set")
	assert.IsType(t, &types.ValidationError{}, err, "should be a ValidationError")

	args = &types.Arguments{
		DataFiles:           []string{"-", "../../testFiles/data/data2.yaml"},
		CommonTemplateFiles: []string{"../../testFiles/common/common1.tmpl"},
		TemplatePaths:       []string{"../../testFiles/templates/template2.tmpl"},
		Output:              "../../testFiles/data/data1.yaml",
		Overwrite:           true,
	}
	err = ValidateArguments(args, parsedFromArgs(t, args), &logger)
	assert.NoError(t, err, "overwrite is set so the file existing is ok")

	args = &types.Arguments{
		DataFiles:     []string{"../../testFiles/data/data2.yaml"},
		TemplatePaths: []string{"../../testFiles/templates/", "../../testFiles/recurse-templates-1/"},
	}
	err = ValidateArguments(args, parsedFromArgs(t, args), &logger)
	assert.NoError(t, err, "valid with recursable template paths")
}
