package config

import (
	"testing"

	"github.com/adam-huganir/yutc/internal/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestValidateArguments(t *testing.T) {
	result, errs := ValidateArguments(
		&types.YutcSettings{
			DataFiles:           []string{"../../testFiles/data/data1.yaml", "../../testFiles/data/data2.yaml"},
			CommonTemplateFiles: []string{"../../testFiles/common/common1.tmpl"},
			TemplatePaths:       []string{"../../testFiles/templates/template1.tmpl", "../../testFiles/templates/template2.tmpl"},
			Output:              "../../testFiles/outputs",
		}, zerolog.Nop())
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, ExitCodeMap["ok"], result, "this is a valid set of inputs")

	result, errs = ValidateArguments(
		&types.YutcSettings{
			DataFiles:     []string{"-"},
			TemplatePaths: []string{"../../testFiles/templates/template1.tmpl"},
			Output:        "-",
		},
		zerolog.Nop(),
	)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, ExitCodeMap["ok"], result, "also valid, only 1 stdin and 1 stdout")
	result, errs = ValidateArguments(
		&types.YutcSettings{
			DataFiles:     []string{"-"},
			TemplatePaths: []string{"-", "../../testFiles/templates/template2.tmpl"},
			Output:        ".",
		},
		zerolog.Nop(),
	)
	assert.NotEqual(t, 0, len(errs))
	assert.Equal(t, ExitCodeMap["cannot use stdin with multiple files"], result, "you can't specify stdin as multiple things")
	result, errs = ValidateArguments(
		&types.YutcSettings{
			DataFiles:           []string{"-", "../../testFiles/data/data2.yaml"},
			CommonTemplateFiles: []string{"../../testFiles/common/common1.tmpl"},
			TemplatePaths:       []string{"../../testFiles/templates/template2.tmpl"},
			Output:              "out.yaml",
		},
		zerolog.Nop(),
	)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, ExitCodeMap["ok"], result, "this is a valid set of inputs")
	result, errs = ValidateArguments(
		&types.YutcSettings{
			DataFiles:           []string{"-", "../../testFiles/data/data2.yaml"},
			CommonTemplateFiles: []string{"../../testFiles/common/common1.tmpl"},
			TemplatePaths:       []string{"../../testFiles/templates/template2.tmpl"},
			Output:              "../../testFiles/data/data1.yaml",
		},
		zerolog.Nop(),
	)
	assert.NotEqual(t, 0, len(errs))
	assert.Equal(t, ExitCodeMap["file exists and `overwrite` is not set"], result, "file exists and overwrite is not set")
	result, errs = ValidateArguments(
		&types.YutcSettings{
			DataFiles:           []string{"-", "../../testFiles/data/data2.yaml"},
			CommonTemplateFiles: []string{"../../testFiles/common/common1.tmpl"},
			TemplatePaths:       []string{"../../testFiles/templates/template2.tmpl"},
			Output:              "../../testFiles/data/data1.yaml",
			Overwrite:           true,
		},
		zerolog.Nop(),
	)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, ExitCodeMap["ok"], result, "overwrite is set so the file existing is ok")
	result, errs = ValidateArguments(
		&types.YutcSettings{
			DataFiles:     []string{"../../testFiles/data/data2.yaml"},
			TemplatePaths: []string{"../../testFiles/templates/", "../../testFiles/recurse-templates-1/"},
		},
		zerolog.Nop(),
	)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, ExitCodeMap["ok"], result, "overwrite is set so the file existing is ok")
}
