package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestMergeData(t *testing.T) {
	// Create temporary data files
	tmpDir := t.TempDir()
	dataFile1 := filepath.Join(tmpDir, "data1.yaml")
	dataFile2 := filepath.Join(tmpDir, "data2.yaml")

	err := os.WriteFile(dataFile1, []byte("key1: value1\nshared: old"), 0o644)
	assert.NoError(t, err)
	err = os.WriteFile(dataFile2, []byte("key2: value2\nshared: new"), 0o644)
	assert.NoError(t, err)

	dataFiles := []*types.DataFileArg{
		{Path: dataFile1},
		{Path: dataFile2},
	}
	logger := zerolog.Nop()

	data, err := MergeData(dataFiles, &logger)
	assert.NoError(t, err)
	assert.Equal(t, "value1", data["key1"])
	assert.Equal(t, "value2", data["key2"])
	assert.Equal(t, "new", data["shared"])
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
