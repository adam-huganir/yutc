package templates

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"text/template"

	"github.com/adam-huganir/yutc/pkg/loader"
	"github.com/stretchr/testify/assert"
)

func TestTemplateInput_ListContainerFiles(t *testing.T) {
	tmpDir := t.TempDir()

	topLevelFile := filepath.Join(tmpDir, "top.txt")
	assert.NoError(t, os.WriteFile(topLevelFile, []byte("root"), 0o644))

	nestedDir := filepath.Join(tmpDir, "nested")
	assert.NoError(t, os.Mkdir(nestedDir, 0o755))

	nestedFile := filepath.Join(nestedDir, "child.txt")
	assert.NoError(t, os.WriteFile(nestedFile, []byte("child"), 0o644))

	ti := NewTemplateInput(tmpDir, false)
	err := ti.CollectContainerChildren()
	assert.NoError(t, err)

	actualPaths := []string{ti.Name}
	for _, child := range ti.AllChildren() {
		actualPaths = append(actualPaths, child.Name)
		assert.Equal(t, loader.SourceKindFile, child.Source)
	}
	sort.Strings(actualPaths)

	expectedPaths := []string{
		loader.NormalizeFilepath(tmpDir),
		loader.NormalizeFilepath(topLevelFile),
		loader.NormalizeFilepath(nestedDir),
		loader.NormalizeFilepath(nestedFile),
	}
	sort.Strings(expectedPaths)

	assert.Equal(t, expectedPaths, actualPaths)
}

func TestTemplateFilenames(t *testing.T) {
	tmpl, err := template.New("test").Parse("{{ .project_name }}")
	assert.NoError(t, err)

	ti := NewTemplateInput("{{ .project_name }}/init.py", false, loader.WithSource(loader.SourceKindFile), loader.WithContentBytes([]byte("content")))
	tis := []*TemplateInput{ti}

	data := map[string]any{"project_name": "my-project"}
	err = TemplateFilenames(tis, tmpl, data)
	assert.NoError(t, err)
	assert.Equal(t, "my-project/init.py", ti.Template.NewName)
}

func TestTemplateFilenames_Error(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse("{{ .project_name }}"))
	tiInvalid := NewTemplateInput("{{ .Unclosed", false, loader.WithSource(loader.SourceKindFile), loader.WithContentBytes([]byte("content")))
	err := TemplateFilenames([]*TemplateInput{tiInvalid}, tmpl, nil)
	assert.Error(t, err)
}

func TestParseTemplateArg(t *testing.T) {
	tests := []struct {
		name                string
		input               string
		expectedPath        string
		expectError         string
		isCommon            bool
		expectedBearerToken string
		expectedBasicAuth   string
	}{
		{
			name:         "simple template path",
			input:        "./my_template.tmpl",
			expectedPath: "my_template.tmpl",
		},
		{
			name:        "template and a jsonpath (error)",
			input:       "jsonpath=.test,src=something.tmpl",
			expectError: "key parameter is not supported for template arguments",
		},
		{
			name:         "common template",
			input:        "./shared.tmpl",
			expectedPath: "shared.tmpl",
			isCommon:     true,
		},
		{
			name:                "template URL with bearer auth",
			input:               "src=https://example.com/template.tmpl,auth=my-secret-token",
			expectedPath:        "https://example.com/template.tmpl",
			expectedBearerToken: "my-secret-token",
		},
		{
			name:              "template URL with basic auth",
			input:             "src=https://example.com/template.tmpl,auth=user:pass",
			expectedPath:      "https://example.com/template.tmpl",
			expectedBasicAuth: "user:pass",
		},
		{
			name:                "common template URL with bearer auth",
			input:               "src=https://example.com/shared.tmpl,auth=token123",
			expectedPath:        "https://example.com/shared.tmpl",
			isCommon:            true,
			expectedBearerToken: "token123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTemplateArg(tt.input, tt.isCommon)

			if tt.expectError != "" {
				assert.Errorf(t, err, "expected error but got none")
				assert.ErrorContains(t, err, tt.expectError)
				return
			}

			assert.NoErrorf(t, err, "unexpected error: %v", err)
			assert.NotNil(t, result)
			assert.Equalf(t, tt.expectedPath, result.Name,
				"expected path %q but got %q", tt.expectedPath, result.Name)
			assert.Equal(t, tt.isCommon, result.IsCommon)
			assert.Equal(t, tt.expectedBearerToken, result.Auth.BearerToken)
			assert.Equal(t, tt.expectedBasicAuth, result.Auth.BasicAuth)
		})
	}
}
