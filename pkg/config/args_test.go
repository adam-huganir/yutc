package config

import (
	"testing"

	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestValidateArguments(t *testing.T) {
	logger := zerolog.Nop()
	err := ValidateArguments(
		&types.Arguments{
			DataFiles:           []string{"../../testFiles/data/data1.yaml", "../../testFiles/data/data2.yaml"},
			CommonTemplateFiles: []string{"../../testFiles/common/common1.tmpl"},
			TemplatePaths:       []string{"../../testFiles/templates/template1.tmpl", "../../testFiles/templates/template2.tmpl"},
			Output:              "../../testFiles/outputs",
		}, &logger)
	assert.NoError(t, err, "this is a valid set of inputs")

	err = ValidateArguments(
		&types.Arguments{
			DataFiles:     []string{"-"},
			TemplatePaths: []string{"../../testFiles/templates/template1.tmpl"},
			Output:        "-",
		},
		&logger,
	)
	assert.NoError(t, err, "also valid, only 1 stdin and 1 stdout")

	err = ValidateArguments(
		&types.Arguments{
			DataFiles:     []string{"-"},
			TemplatePaths: []string{"-", "../../testFiles/templates/template2.tmpl"},
			Output:        ".",
		},
		&logger,
	)
	assert.Error(t, err, "you can't specify stdin as multiple things")
	assert.IsType(t, &types.ValidationError{}, err, "should be a ValidationError")

	err = ValidateArguments(
		&types.Arguments{
			DataFiles:           []string{"-", "../../testFiles/data/data2.yaml"},
			CommonTemplateFiles: []string{"../../testFiles/common/common1.tmpl"},
			TemplatePaths:       []string{"../../testFiles/templates/template2.tmpl"},
			Output:              "out.yaml",
		},
		&logger,
	)
	assert.NoError(t, err, "this is a valid set of inputs")

	err = ValidateArguments(
		&types.Arguments{
			DataFiles:           []string{"-", "../../testFiles/data/data2.yaml"},
			CommonTemplateFiles: []string{"../../testFiles/common/common1.tmpl"},
			TemplatePaths:       []string{"../../testFiles/templates/template2.tmpl"},
			Output:              "../../testFiles/data/data1.yaml",
		},
		&logger,
	)
	assert.Error(t, err, "file exists and overwrite is not set")
	assert.IsType(t, &types.ValidationError{}, err, "should be a ValidationError")

	err = ValidateArguments(
		&types.Arguments{
			DataFiles:           []string{"-", "../../testFiles/data/data2.yaml"},
			CommonTemplateFiles: []string{"../../testFiles/common/common1.tmpl"},
			TemplatePaths:       []string{"../../testFiles/templates/template2.tmpl"},
			Output:              "../../testFiles/data/data1.yaml",
			Overwrite:           true,
		},
		&logger,
	)
	assert.NoError(t, err, "overwrite is set so the file existing is ok")

	err = ValidateArguments(
		&types.Arguments{
			DataFiles:     []string{"../../testFiles/data/data2.yaml"},
			TemplatePaths: []string{"../../testFiles/templates/", "../../testFiles/recurse-templates-1/"},
		},
		&logger,
	)
	assert.NoError(t, err, "valid with recursable template paths")
}
