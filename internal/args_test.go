package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidateArguments(t *testing.T) {
	assert.Equal(t, int64(0), ValidateArguments(
		&CLISettings{
			DataFiles:           []string{"../testFiles/data/data1.yaml", "../testFiles/data/data2.yaml"},
			CommonTemplateFiles: []string{"../testFiles/common/common1.tmpl"},
			TemplatePaths:       []string{"../testFiles/templates/template1.tmpl", "../testFiles/templates/template2.tmpl"},
			Output:              "../testFiles/outputs",
		},
	), "this is a valid set of inputs")
	assert.Equal(t, int64(0), ValidateArguments(
		&CLISettings{
			DataFiles:     []string{"-"},
			TemplatePaths: []string{"../testFiles/templates/template1.tmpl"},
			Output:        "-",
		},
	), "also valid, only 1 stdin and 1 stdout")
	assert.Equal(t, exitCodeMap["cannot use stdin with multiple files"], ValidateArguments(
		&CLISettings{
			DataFiles:     []string{"-"},
			TemplatePaths: []string{"-", "../testFiles/templates/template2.tmpl"},
			Output:        ".",
		},
	), "you can't specify stdin as multiple things")
	assert.Equal(t, int64(0), ValidateArguments(
		&CLISettings{
			DataFiles:           []string{"-", "../testFiles/data/data2.yaml"},
			CommonTemplateFiles: []string{"../testFiles/common/common1.tmpl"},
			TemplatePaths:       []string{"../testFiles/templates/template2.tmpl"},
			Output:              "out.yaml",
		},
	), "this is a valid set of inputs")
	assert.Equal(t, exitCodeMap["file exists and `overwrite` is not set"], ValidateArguments(
		&CLISettings{
			DataFiles:           []string{"-", "../testFiles/data/data2.yaml"},
			CommonTemplateFiles: []string{"../testFiles/common/common1.tmpl"},
			TemplatePaths:       []string{"../testFiles/templates/template2.tmpl"},
			Output:              "../testFiles/data/data1.yaml",
		},
	), "file exists and overwrite is not set")
	assert.Equal(t, int64(0), ValidateArguments(
		&CLISettings{
			DataFiles:           []string{"-", "../testFiles/data/data2.yaml"},
			CommonTemplateFiles: []string{"../testFiles/common/common1.tmpl"},
			TemplatePaths:       []string{"../testFiles/templates/template2.tmpl"},
			Output:              "../testFiles/data/data1.yaml",
			Overwrite:           true,
		},
	), "overwrite is set so the file existing is ok")
}
