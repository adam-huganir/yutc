package data

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/adam-huganir/yutc/pkg/util"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/theory/jsonpath"
)

func TestMergeData(t *testing.T) {
	tests := []struct {
		name         string
		fileContents map[string]string
		helmMode     bool
		expectedData map[string]any
		expectError  bool
	}{
		{
			name: "default merge",
			fileContents: map[string]string{
				"data1.yaml": util.MustDedent(`
									key1: value1
									shared: old
									key2: value2`),
				"data2.yaml": util.MustDedent(`
									key1: value1
									shared: new
									key2: value2`),
			},
			helmMode: false,
			expectedData: map[string]any{
				"key1":   "value1",
				"key2":   "value2",
				"shared": "new",
			},
			expectError: false,
		},
		{
			name: "merge yaml with toml and json",
			fileContents: map[string]string{
				"data1.yaml": util.MustDedent(`
									key1: value1
									shared: old
									key2: value2`),
				"data2.toml": util.MustDedent(`
									key1 = "value1"
									shared = "new"
									key2 = "value2"`),
				"data3.json": util.MustDedent(`
									{
										"key2": "value2 but different"
									}`),
			},
			helmMode: false,
			expectedData: map[string]any{
				"key1":   "value1",
				"key2":   "value2 but different",
				"shared": "new",
			},
			expectError: false,
		},
		{
			name: "invalid yaml",
			fileContents: map[string]string{
				"data1.yaml": util.MustDedent(`
									key1: value1
									shared: old
									key2 = value2`),
			},
			helmMode:     false,
			expectedData: nil,
			expectError:  true,
		},
		{
			name: "invalid toml",
			fileContents: map[string]string{
				"data1.toml": util.MustDedent(`
									key1 = "value1"
									shared = "old"
									key2 : "value2"`),
			},
			helmMode:     false,
			expectedData: nil,
			expectError:  true,
		},
		{
			name: "invalid json",
			fileContents: map[string]string{
				"data1.json": util.MustDedent(`
									{
										"key1": "value1",
										"shared": "old",
									}`),
			},
			helmMode:     false,
			expectedData: nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			var dataFiles []*FileArg

			// Get keys and sort them to ensure deterministic file processing order
			var filenames []string
			for filename := range tt.fileContents {
				filenames = append(filenames, filename)
			}
			sort.Strings(filenames)

			for _, filename := range filenames {
				content := tt.fileContents[filename]
				filePath := filepath.Join(tmpDir, filename)
				err := os.WriteFile(filePath, []byte(content), 0o644)
				assert.NoError(t, err)
				fk := FileKind("data")
				fa := NewFileArgFile(filePath, &fk)
				dataFiles = append(dataFiles, &fa)
			}

			logger := zerolog.Nop()
			data, err := MergeDataFiles(dataFiles, tt.helmMode, &logger)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedData, data)
			}
		})
	}
}

func TestFileArg_ListContainerFiles(t *testing.T) {
	tmpDir := t.TempDir()

	topLevelFile := filepath.Join(tmpDir, "top.txt")
	assert.NoError(t, os.WriteFile(topLevelFile, []byte("root"), 0o644))

	nestedDir := filepath.Join(tmpDir, "nested")
	assert.NoError(t, os.Mkdir(nestedDir, 0o755))

	nestedFile := filepath.Join(nestedDir, "child.txt")
	assert.NoError(t, os.WriteFile(nestedFile, []byte("child"), 0o644))

	fileArg := NewFileArgFile(tmpDir, nil)
	err := fileArg.CollectContainerChildren()
	assert.NoError(t, err)

	actualPaths := []string{fileArg.Name}
	for _, fa := range fileArg.AllChildren() {
		actualPaths = append(actualPaths, fa.Name)
		assert.Equal(t, "file", fa.Source)
	}
	sort.Strings(actualPaths)

	expectedPaths := []string{
		NormalizeFilepath(tmpDir),
		NormalizeFilepath(topLevelFile),
		NormalizeFilepath(nestedDir),
		NormalizeFilepath(nestedFile),
	}
	sort.Strings(expectedPaths)

	assert.Equal(t, expectedPaths, actualPaths)
}

func TestMergeDataWithKeys(t *testing.T) {
	tests := []struct {
		name         string
		setupFiles   map[string]string
		dataFileArgs []*FileArg
		helmMode     bool
		expectedData map[string]any
		expectError  bool
	}{
		{
			name: "nest Chart data without helm mode",
			setupFiles: map[string]string{
				"chart.yaml": util.MustDedent(`
									name: my-chart
									version: 1.0.0
									description: a chart`),
			},
			dataFileArgs: []*FileArg{
				{Name: "chart.yaml", JSONPath: jsonpath.MustParse("$.Chart")},
			},
			helmMode: false,
			expectedData: map[string]any{
				"Chart": map[string]any{
					"name":        "my-chart",
					"version":     "1.0.0",
					"description": "a chart",
				},
			},
			expectError: false,
		},
		{
			name: "nest Chart data with helm mode (to go struct casing)",
			setupFiles: map[string]string{
				"chart.yaml": util.MustDedent(`
									name: my-chart
									version: 1.0.0
									description: a chart`),
			},
			dataFileArgs: []*FileArg{
				{Name: "chart.yaml", JSONPath: jsonpath.MustParse("$.Chart")},
			},
			helmMode: true,
			expectedData: map[string]any{
				"Chart": map[string]any{
					"Name":        "my-chart",
					"Version":     "1.0.0",
					"Description": "a chart",
				},
			},
			expectError: false,
		},
		{
			name: "nest data by path",
			setupFiles: map[string]string{
				"chart.yaml": util.MustDedent(`
									name: my-chart
									version: 1.0.0
									description: a chart`),
			},
			dataFileArgs: []*FileArg{
				{Name: "chart.yaml", JSONPath: jsonpath.MustParse("$.some.path.to[0].chart")},
			},
			helmMode: false,
			expectedData: map[string]any{
				"some": map[string]any{
					"path": map[string]any{
						"to": []any{
							map[string]any{
								"chart": map[string]any{
									"description": "a chart",
									"name":        "my-chart",
									"version":     "1.0.0",
								},
							},
						},
					}},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Get keys and sort them to ensure deterministic file creation order
			var filenames []string
			for filename := range tt.setupFiles {
				filenames = append(filenames, filename)
			}
			sort.Strings(filenames)

			// Create data for the current test case
			for _, filename := range filenames {
				content := tt.setupFiles[filename]
				filePath := filepath.Join(tmpDir, filename)
				err := os.WriteFile(filePath, []byte(content), 0o644)
				assert.NoError(t, err)
			}

			// Prepare dataFileArgs with actual temporary file paths
			var currentDataFileArgs []*FileArg
			for _, dfa := range tt.dataFileArgs {
				actualPath := filepath.Join(tmpDir, dfa.Name)
				fk := FileKind("data")
				fa := NewFileArgFile(actualPath, &fk)
				fa.JSONPath = dfa.JSONPath
				currentDataFileArgs = append(currentDataFileArgs, &fa)
			}

			logger := zerolog.Nop()
			data, err := MergeDataFiles(currentDataFileArgs, tt.helmMode, &logger)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedData, data)
			}
		})
	}
}

func TestLoadDataFiles(t *testing.T) {
	tmpDir := t.TempDir()
	dataFile := filepath.Join(tmpDir, "data.yaml")
	err := os.WriteFile(dataFile, []byte("key: value"), 0o644)
	assert.NoError(t, err)

	//dataFiles := []*FileArg{
	//	{Name: dataFile},
	//}
	//logger := zerolog.Nop()

	//loadedFiles, err := LoadFiles(dataFiles, tmpDir, &logger)
	//assert.NoError(t, err)
	//assert.Len(t, loadedFiles, 1)
	//assert.Equal(t, dataFile, loadedFiles[0].Name)
}

func TestLoadTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	tmplFile := filepath.Join(tmpDir, "template.tmpl")
	err := os.WriteFile(tmplFile, []byte("{{ .key }}"), 0o644)
	assert.NoError(t, err)

	//templatePaths := []string{tmplFile}
	//logger := zerolog.Nop()

	//loadedTemplates, err := LoadTemplates(templatePaths, tmpDir, &logger)
	//assert.NoError(t, err)
	//assert.Len(t, loadedTemplates, 1)
	//assert.Equal(t, tmplFile, loadedTemplates[0])
}
