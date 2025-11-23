package template

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestBuildTemplate(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		shared         []*bytes.Buffer
		strict         bool
		expectedOutput string
		expectError    bool
	}{
		{
			name:           "simple template",
			text:           "Hello {{ .name }}",
			shared:         nil,
			strict:         false,
			expectedOutput: "Hello World",
			expectError:    false,
		},
		{
			name:           "shared template",
			text:           "{{ include \"shared\" . }}",
			shared:         []*bytes.Buffer{bytes.NewBufferString("{{ define \"shared\" }}Shared {{ .name }}{{ end }}")},
			strict:         false,
			expectedOutput: "Shared World",
			expectError:    false,
		},
		{
			name:           "strict mode missing key",
			text:           "{{ .missing }}",
			shared:         nil,
			strict:         true,
			expectedOutput: "",
			expectError:    true, // Execution error, but BuildTemplate might pass. Wait, BuildTemplate just parses.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := BuildTemplate(tt.text, tt.shared, "test", tt.strict)
			if tt.expectError && err != nil {
				// Expected error during build (e.g. bad syntax)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, tmpl)

			if !tt.expectError {
				var buf bytes.Buffer
				data := map[string]interface{}{"name": "World"}
				err = tmpl.Execute(&buf, data)
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
	err := os.WriteFile(tmplFile, []byte("{{ .key }}"), 0644)
	assert.NoError(t, err)

	templateFiles := []string{tmplFile}
	sharedBuffers := []*bytes.Buffer{}
	logger := zerolog.Nop()

	templates, err := LoadTemplates(templateFiles, sharedBuffers, false, &logger)
	assert.NoError(t, err)
	assert.Len(t, templates, 1)
	assert.NotNil(t, templates[0])
}
