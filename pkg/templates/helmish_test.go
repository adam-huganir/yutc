package templates

import (
	"bytes"
	"testing"

	"github.com/adam-huganir/yutc/pkg/data"
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
		want func(string, any) (string, error)
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
			tmpl, err := InitTemplate([]*data.FileArg{
				data.NewFileArgWithContent(
					"file",
					data.FileKindTemplate,
					"file",
					[]byte(tt.args.templateB),
				),
			}, false)
			assert.NoError(t, err)
			args := []*data.FileArg{data.NewFileArgWithContent(tt.name, data.FileKindTemplate, "file", []byte(tt.args.templateA))}
			tmpl, err = ParseTemplateItems(tmpl, args)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			outData := new(bytes.Buffer)
			err = tmpl.ExecuteTemplate(outData, tt.name, map[string]any{"target": "World"})
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
		want func(string, any) (string, error)
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
			tmpl, err := InitTemplate(nil, false)
			assert.NoError(t, err)
			args := []*data.FileArg{data.NewFileArgWithContent(tt.name, data.FileKindTemplate, "file", []byte(tt.args.templateA))}
			tmpl, err = ParseTemplateItems(tmpl, args)
			assert.NoError(t, err)
			outData := new(bytes.Buffer)
			err = tmpl.ExecuteTemplate(outData, tt.name, map[string]any{"text": "Hello World"})
			assert.NoError(t, err)
			assert.Equal(t, outData.String(), "watch me say Hello World")
		})
	}
}
