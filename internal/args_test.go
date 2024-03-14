package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidateArguments(t *testing.T) {
	type args struct {
		dataFiles       []string
		sharedTemplates []string
		templateFiles   []string
		output          string
		overwrite       bool
	}
	assert.Equal(t, int64(0), ValidateArguments(
		[]string{"../testData/inputData/data1.yaml", "../testData/inputData/data2.yaml"},
		[]string{"../testData/testTemplates/common1.tmpl"},
		[]string{"../testData/testTemplates/template1.tmpl", "../testData/testTemplates/template2.tmpl"},
		"../testData/outputs",
		false,
	), "this is a valid set of inputs")
	assert.Equal(t, int64(0), ValidateArguments(
		[]string{"-"},
		nil,
		[]string{"../testData/testTemplates/template1.tmpl"},
		"-",
		false,
	), "also valid, only 1 stdin and 1 stdout")
	assert.Equal(t, int64(64), ValidateArguments(
		[]string{"-"},
		nil,
		[]string{"-", "../testData/testTemplates/template2.tmpl"},
		".",
		false,
	), "you can't specify stdin as multiple things")
	assert.Equal(t, int64(0), ValidateArguments(
		[]string{"-", "../testData/inputData/data2.yaml"},
		[]string{"../testData/testTemplates/common1.tmpl"},
		[]string{"../testData/testTemplates/template2.tmpl"},
		"out.yaml",
		false,
	), "this is a valid set of inputs")
	assert.Equal(t, int64(16), ValidateArguments(
		[]string{"-", "../testData/inputData/data2.yaml"},
		[]string{"../testData/testTemplates/common1.tmpl"},
		[]string{"../testData/testTemplates/template2.tmpl"},
		"../testData/inputData/data1.yaml",
		false,
	), "file exists and overwrite is not set")
	assert.Equal(t, int64(0), ValidateArguments(
		[]string{"-", "../testData/inputData/data2.yaml"},
		[]string{"../testData/testTemplates/common1.tmpl"},
		[]string{"../testData/testTemplates/template2.tmpl"},
		"../testData/inputData/data1.yaml",
		true,
	), "overwrite is set so the file existing is ok")
}
