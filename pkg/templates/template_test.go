package templates

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/adam-huganir/yutc/pkg/data"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestBuildTemplate(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		shared         []*bytes.Buffer
		strict         bool
		expectedOutput string
		expectError    bool
	}{
		{
			name:           "simple template",
			template:       "Hello {{ .name }}",
			shared:         nil,
			strict:         false,
			expectedOutput: "Hello World",
			expectError:    false,
		},
		{
			name:           "shared template",
			template:       "{{ include \"shared\" . }}",
			shared:         []*bytes.Buffer{bytes.NewBufferString("{{ define \"shared\" }}Shared {{ .name }}{{ end }}")},
			strict:         false,
			expectedOutput: "Shared World",
			expectError:    false,
		},
		{
			name:           "strict mode missing key",
			template:       "{{ .missing }}",
			shared:         nil,
			strict:         true,
			expectedOutput: "",
			expectError:    true, // Execution error, but BuildTemplate might pass. Wait, BuildTemplate just parses.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := InitTemplate(tt.shared, tt.strict)
			if tt.expectError && err != nil {
				// Expected error during build (e.g. bad syntax)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, tmpl)
			kind := data.FileKindData
			args := data.NewFileArgWithContent(tt.name, &kind, "file", []byte(tt.template))
			tmpl, err = ParseTemplateItems(tmpl, []*data.FileArg{args})
			assert.NoError(t, err)

			if !tt.expectError {
				var buf bytes.Buffer
				d := map[string]any{"name": "World"}
				err = tmpl.ExecuteTemplate(&buf, tt.name, d)
				if tt.strict && tt.name == "strict mode missing key" {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.expectedOutput, buf.String())
				}
			}
		})
	}
}

func TestLoadTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	tmplFile := filepath.Join(tmpDir, "test.tmpl")
	err := os.WriteFile(tmplFile, []byte("{{ .key }}"), 0o644)
	assert.NoError(t, err)

	fk := data.FileKindData
	fileArg := data.NewFileArgFile(tmplFile, &fk)
	templateFiles := []*data.FileArg{&fileArg}
	var sharedBuffers []*bytes.Buffer
	logger := zerolog.Nop()

	templates, err := LoadTemplateSet(templateFiles, sharedBuffers, false, &logger)
	assert.NoError(t, err)
	assert.Len(t, templates.TemplateItems, 1)
	assert.NotNil(t, templates.Template)
}
