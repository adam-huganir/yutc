package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/adam-huganir/yutc/pkg/util"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
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
			var dataFiles []*types.DataFileArg

			for filename, content := range tt.fileContents {
				filePath := filepath.Join(tmpDir, filename)
				err := os.WriteFile(filePath, []byte(content), 0o644)
				assert.NoError(t, err)
				dataFiles = append(dataFiles, &types.DataFileArg{Path: filePath})
			}

			logger := zerolog.Nop()
			data, err := MergeData(dataFiles, tt.helmMode, &logger)

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

	dataFiles := []*types.DataFileArg{
		{Path: dataFile},
	}
	logger := zerolog.Nop()

	loadedFiles, err := LoadDataFiles(dataFiles, tmpDir, &logger)
	assert.NoError(t, err)
	assert.Len(t, loadedFiles, 1)
	assert.Equal(t, dataFile, loadedFiles[0].Path)
}

func TestLoadTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	tmplFile := filepath.Join(tmpDir, "template.tmpl")
	err := os.WriteFile(tmplFile, []byte("{{ .key }}"), 0o644)
	assert.NoError(t, err)

	templatePaths := []string{tmplFile}
	logger := zerolog.Nop()

	loadedTemplates, err := LoadTemplates(templatePaths, tmpDir, &logger)
	assert.NoError(t, err)
	assert.Len(t, loadedTemplates, 1)
	assert.Equal(t, tmplFile, loadedTemplates[0])
}
