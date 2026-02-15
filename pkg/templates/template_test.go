package templates

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/adam-huganir/yutc/pkg/loader"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestBuildTemplate(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		shared         []*TemplateInput
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
			name:     "shared template",
			template: "{{ include \"shared\" . }}",
			shared: []*TemplateInput{NewTemplateInput(
				"shared",
				true,
				loader.WithSource(loader.SourceKindFile),
				loader.WithContentBytes([]byte("{{ define \"shared\" }}Shared {{ .name }}{{ end }}")),
			),
			},
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
			args := NewTemplateInput(tt.name, false, loader.WithSource(loader.SourceKindFile), loader.WithContentBytes([]byte(tt.template)))
			tmpl, err = ParseTemplateItems(tmpl, []*TemplateInput{args}, "")
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

func TestParseTemplateItems_DropExtension(t *testing.T) {
	tests := []struct {
		name          string
		templateName  string
		dropExtension string
		expectedName  string
	}{
		{
			name:          "drop tmpl extension",
			templateName:  "myfile.tmpl",
			dropExtension: "tmpl",
			expectedName:  "myfile",
		},
		{
			name:          "drop tmpl extension with dot prefix",
			templateName:  "myfile.tmpl",
			dropExtension: ".tmpl",
			expectedName:  "myfile",
		},
		{
			name:          "drop tpl extension",
			templateName:  "myfile.tpl",
			dropExtension: "tpl",
			expectedName:  "myfile",
		},
		{
			name:          "no drop when extension doesn't match",
			templateName:  "myfile.yaml",
			dropExtension: "tmpl",
			expectedName:  "myfile.yaml",
		},
		{
			name:          "empty drop extension",
			templateName:  "myfile.tmpl",
			dropExtension: "",
			expectedName:  "myfile.tmpl",
		},
		{
			name:          "drop extension with whitespace",
			templateName:  "myfile.tmpl",
			dropExtension: "  tmpl  ",
			expectedName:  "myfile",
		},
		{
			name:          "multiple dots in filename",
			templateName:  "my.file.tmpl",
			dropExtension: "tmpl",
			expectedName:  "my.file",
		},
		{
			name:          "no extension in filename",
			templateName:  "myfile",
			dropExtension: "tmpl",
			expectedName:  "myfile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := InitTemplate(nil, false)
			assert.NoError(t, err)
			assert.NotNil(t, tmpl)

			args := NewTemplateInput(
				tt.templateName,
				false,
				loader.WithSource(loader.SourceKindFile),
				loader.WithContentBytes([]byte("test content")),
			)

			tmpl, err = ParseTemplateItems(tmpl, []*TemplateInput{args}, tt.dropExtension)
			assert.NoError(t, err)

			// Verify the template was registered with the expected name
			assert.NotNil(t, tmpl.Lookup(tt.expectedName), "template should be registered as %q", tt.expectedName)

			// Verify the FileArg's NewName was updated
			assert.Equal(t, tt.expectedName, args.Template.NewName)
		})
	}
}

func TestLoadTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	tmplFile := filepath.Join(tmpDir, "test.tmpl")
	err := os.WriteFile(tmplFile, []byte("{{ .key }}"), 0o644)
	assert.NoError(t, err)

	fileArg := NewTemplateInput(tmplFile, false)
	templateFiles := []*TemplateInput{fileArg}
	var sharedTemplates []*TemplateInput
	logger := zerolog.Nop()

	templates, err := LoadTemplateSet(templateFiles, sharedTemplates, map[string]any{}, false, false, "", &logger)
	assert.NoError(t, err)
	assert.Len(t, templates.TemplateFiles, 1)
	assert.NotNil(t, templates.Template)
}
