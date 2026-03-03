package templates

import (
	"bytes"
	"runtime"
	"testing"

	"github.com/adam-huganir/yutc/pkg/loader"
	"github.com/stretchr/testify/assert"
)

func echoCmd(s string) string {
	if runtime.GOOS == "windows" {
		return "Write-Host -NoNewline '" + s + "'"
	}
	return "printf '" + s + "'"
}

func TestShell(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		expectedOutput string
		wantErr        bool
	}{
		{
			name:           "simple echo",
			command:        echoCmd("hello"),
			expectedOutput: "hello",
		},
		{
			name:    "command not found",
			command: "this_command_does_not_exist_xyz",
			wantErr: true,
		},
		{
			name:    "non-zero exit code",
			command: "exit 1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := Shell(tt.command)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, out)
		})
	}

	t.Run("multiline output", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("multiline printf not supported on Windows PowerShell without escaping")
		}
		out, err := Shell("printf 'a\nb'")
		assert.NoError(t, err)
		assert.Equal(t, "a\nb", out)
	})
}

func TestShellTemplateFunction(t *testing.T) {
	tests := []struct {
		name           string
		tmplText       string
		expectedOutput string
		wantErr        bool
		allowShell     bool
	}{
		{
			name:           "shell func available when allowed",
			allowShell:     true,
			tmplText:       `{{ shell "echo hello" }}`,
			expectedOutput: "hello",
		},
		{
			name:       "shell func not registered when not allowed",
			allowShell: false,
			tmplText:   `{{ shell "echo hello" }}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := InitTemplate(nil, false, tt.allowShell)
			assert.NoError(t, err)

			args := []*Input{{
				FileEntry: &loader.FileEntry{
					Source:  loader.SourceKindFile,
					Name:    tt.name,
					Content: &loader.FileContent{Data: []byte(tt.tmplText), Read: true},
				},
			}}
			tmpl, err = ParseTemplateItems(tmpl, args, "")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			var buf bytes.Buffer
			err = tmpl.ExecuteTemplate(&buf, tt.name, nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, buf.String())
		})
	}
}
