package template

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncludeFun(t *testing.T) {
	type args struct {
		templateA string
		templateB string
	}
	tests := []struct {
		name string
		args args
		want func(string, interface{}) (string, error)
	}{
		{
			name: "Test include",
			args: args{
				templateA: "watch me say {{ include \"templateB\" . }}",
				templateB: `{{- define "templateB" }}Hello {{.target}}{{- end }}`,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := BuildTemplate(tt.args.templateA, nil, "templateA", false)
			if err != nil {
				t.Errorf("BuildTemplate() = %v, want %v", err, nil)
			}
			tmpl, err = tmpl.Parse(tt.args.templateB)
			if err != nil {
				t.Errorf("Parse() = %v, want %v", err, nil)
			}
			outData := new(bytes.Buffer)
			err = tmpl.Execute(outData, map[string]any{"target": "World"})
			assert.NoError(t, err)
			assert.Equal(t, outData.String(), "watch me say Hello World")
		})
	}
}

func TestTplFun(t *testing.T) {
	type args struct {
		templateA string
	}
	tests := []struct {
		name string
		args args
		want func(string, interface{}) (string, error)
	}{
		{
			name: "Test tpl",
			args: args{
				templateA: `
					{{- $a := "say {{ .text }}" -}}
					watch me {{ tpl $a . -}}
				`,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := BuildTemplate(tt.args.templateA, nil, "templateA", false)
			if err != nil {
				t.Errorf("BuildTemplate() = %v, want %v", err, nil)
			}
			outData := new(bytes.Buffer)
			err = tmpl.Execute(outData, map[string]any{"text": "Hello World"})
			assert.NoError(t, err)
			assert.Equal(t, outData.String(), "watch me say Hello World")
		})
	}
}
