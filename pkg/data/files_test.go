package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestCountDataRecursables(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0o755)
	assert.NoError(t, err)

	file1 := filepath.Join(tempDir, "file1.txt")
	err = os.WriteFile(file1, []byte("content"), 0o644)
	assert.NoError(t, err)

	diDir := NewInput(subDir, nil)
	diFile := NewInput(file1, nil)

	count, err := CountDataRecursables([]*Input{diDir, diFile})
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Test URL archive (mocked by extension)
	diURL := NewInput("http://example.com/test.zip", []FileEntryOption{WithSource(SourceKindURL)})
	count, err = CountDataRecursables([]*Input{diURL})
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestResolveDataPaths_Complex(t *testing.T) {
	tempDir := t.TempDir()

	// Single file
	file1 := filepath.Join(tempDir, "file1.yaml")
	err := os.WriteFile(file1, []byte("key: value"), 0o644)
	assert.NoError(t, err)

	logger := zerolog.Nop()
	outFiles, err := ResolveDataPaths([]string{file1}, &logger)
	assert.NoError(t, err)
	assert.Len(t, outFiles, 1)

	// Directory
	subDir := filepath.Join(tempDir, "mysubdir")
	err = os.Mkdir(subDir, 0o755)
	assert.NoError(t, err)
	file2 := filepath.Join(subDir, "file2.yaml")
	err = os.WriteFile(file2, []byte("key2: value2"), 0o644)
	assert.NoError(t, err)

	outFiles, err = ResolveDataPaths([]string{subDir}, &logger)
	assert.NoError(t, err)
	assert.True(t, len(outFiles) >= 1)

	// Error path: non-existent file
	_, err = ResolveDataPaths([]string{filepath.Join(tempDir, "nonexistent.yaml")}, &logger)
	assert.Error(t, err)
}

func TestMakeDirExist_Error(t *testing.T) {
	tempFile, err := os.CreateTemp("", "mkdir-error-test")
	assert.NoError(t, err)
	_ = os.Remove(tempFile.Name())
}
