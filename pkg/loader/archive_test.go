package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsArchive(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{"tar.gz", "test.tar.gz", true},
		{"tgz", "test.tgz", true},
		{"tar", "test.tar", true},
		{"zip", "test.zip", true},
		{"gz", "test.gz", true},
		{"txt", "test.txt", false},
		{"yaml", "test.yaml", false},
		{"no extension", "test", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsArchive(tt.filePath))
		})
	}
}

func TestReadTar(t *testing.T) {
	// Find the project root to locate testFiles
	wd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Join(wd, "..", "..")
	tarGzPath := filepath.Join(projectRoot, "testFiles", "poetry-init", "from-dir.tar.gz")

	t.Run("valid tar.gz", func(t *testing.T) {
		files, err := ReadTar(tarGzPath)
		require.NoError(t, err)
		assert.NotEmpty(t, files)

		// Check for some expected files in the archive
		foundPyProject := false
		for _, f := range files {
			if filepath.Base(f.FilePath) == "pyproject.toml" {
				foundPyProject = true
				assert.NotEmpty(t, f.Data)
			}
		}
		assert.True(t, foundPyProject, "pyproject.toml should be in the archive")
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := ReadTar("non-existent.tar.gz")
		assert.Error(t, err)
	})
}

func TestReadZip(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Join(wd, "..", "..")
	zipPath := filepath.Join(projectRoot, "testFiles", "poetry-init", "from-dir.zip")

	t.Run("valid zip", func(t *testing.T) {
		files, err := ReadZip(zipPath)
		require.NoError(t, err)
		assert.NotEmpty(t, files)

		foundPyProject := false
		for _, f := range files {
			if filepath.Base(f.FilePath) == "pyproject.toml" {
				foundPyProject = true
				assert.NotEmpty(t, f.Data)
			}
		}
		assert.True(t, foundPyProject, "pyproject.toml should be in the archive")
	})
}

func TestReadArchive(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Join(wd, "..", "..")
	tarGzPath := filepath.Join(projectRoot, "testFiles", "poetry-init", "from-dir.tar.gz")
	zipPath := filepath.Join(projectRoot, "testFiles", "poetry-init", "from-dir.zip")

	t.Run("dispatch tar.gz", func(t *testing.T) {
		files, err := ReadArchive(tarGzPath)
		require.NoError(t, err)
		assert.NotEmpty(t, files)
	})

	t.Run("dispatch zip", func(t *testing.T) {
		files, err := ReadArchive(zipPath)
		require.NoError(t, err)
		assert.NotEmpty(t, files)
	})
}
