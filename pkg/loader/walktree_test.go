package loader

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEntries(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Join(wd, "..", "..")

	t.Run("directory", func(t *testing.T) {
		dirPath := filepath.Join(projectRoot, "testFiles", "common")
		fe := NewFileEntry(dirPath)
		entries, err := GetEntries(fe, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 2)

		found1 := false
		found2 := false
		for _, e := range entries {
			if filepath.Base(e.Name) == "common1.tmpl" {
				found1 = true
			}
			if filepath.Base(e.Name) == "common2.tmpl" {
				found2 = true
			}
		}
		assert.True(t, found1)
		assert.True(t, found2)
	})

	t.Run("tar.gz archive", func(t *testing.T) {
		archivePath := filepath.Join(projectRoot, "testFiles", "poetry-init", "from-dir.tar.gz")
		fe := NewFileEntry(archivePath)
		entries, err := GetEntries(fe, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, entries)

		found := false
		for _, e := range entries {
			// Check if it's our synthetic name
			assert.Contains(t, e.Name, "#")
			if filepath.Base(e.Name) == "pyproject.toml" {
				found = true
				assert.True(t, e.Content.Read)
				assert.NotEmpty(t, e.Content.Data)
			}
		}
		assert.True(t, found)
	})

	t.Run("zip archive", func(t *testing.T) {
		archivePath := filepath.Join(projectRoot, "testFiles", "poetry-init", "from-dir.zip")
		fe := NewFileEntry(archivePath)
		entries, err := GetEntries(fe, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, entries)

		found := false
		for _, e := range entries {
			assert.Contains(t, e.Name, "#")
			if filepath.Base(e.Name) == "pyproject.toml" {
				found = true
				assert.True(t, e.Content.Read)
				assert.NotEmpty(t, e.Content.Data)
			}
		}
		assert.True(t, found)
	})

	t.Run("gzip file", func(t *testing.T) {
		// Create a temporary .gz file for testing
		tmpDir := t.TempDir()
		gzPath := filepath.Join(tmpDir, "test.txt.gz")
		f, err := os.Create(gzPath)
		require.NoError(t, err)

		gw := gzip.NewWriter(f)
		_, err = gw.Write([]byte("hello world"))
		require.NoError(t, err)
		require.NoError(t, gw.Close())
		require.NoError(t, f.Close())

		fe := NewFileEntry(gzPath)
		entries, err := GetEntries(fe, nil)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Contains(t, entries[0].Name, "#test.txt")
		assert.Equal(t, []byte("hello world"), entries[0].Content.Data)
	})

	t.Run("in-memory archive", func(t *testing.T) {
		// Mock a loaded archive (e.g. from URL)
		archivePath := "https://example.com/test.zip"

		// Create a real zip in memory
		buf := new(bytes.Buffer)
		zw := zip.NewWriter(buf)
		f, err := zw.Create("hello.txt")
		require.NoError(t, err)
		_, err = f.Write([]byte("hi"))
		require.NoError(t, err)
		require.NoError(t, zw.Close())

		fe := NewFileEntry(archivePath, WithContentBytes(buf.Bytes()))
		fe.Content.Filename = "test.zip" // Set by ReadURL normally
		fe.Content.Mimetype = "application/zip"

		entries, err := GetEntries(fe, nil)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Contains(t, entries[0].Name, "test.zip#hello.txt")
		assert.Equal(t, []byte("hi"), entries[0].Content.Data)
	})

	t.Run("nested archive", func(t *testing.T) {
		// Create a zip containing a tar.gz
		tmpDir := t.TempDir()

		// 1. Create inner tar.gz
		innerTarBuf := new(bytes.Buffer)
		tw := tar.NewWriter(innerTarBuf)
		hdr := &tar.Header{
			Name: "inner.txt",
			Mode: 0o600,
			Size: int64(len("inner content")),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err = tw.Write([]byte("inner content"))
		require.NoError(t, tw.Close())

		// 2. Create outer zip
		outerZipPath := filepath.Join(tmpDir, "outer.zip")
		f, err := os.Create(outerZipPath)
		require.NoError(t, err)
		zw := zip.NewWriter(f)
		zf, err := zw.Create("nested.tar")
		require.NoError(t, err)
		_, err = zf.Write(innerTarBuf.Bytes())
		require.NoError(t, err)
		require.NoError(t, zw.Close())
		require.NoError(t, f.Close())

		// 3. Test GetEntries (first level)
		fe := NewFileEntry(outerZipPath)
		entries, err := GetEntries(fe, nil)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Contains(t, entries[0].Name, "outer.zip#nested.tar")

		// 4. Test GetEntries (second level - the nested archive)
		nestedFe := entries[0]
		isArchive, err := nestedFe.IsArchive()
		require.NoError(t, err)
		assert.True(t, isArchive)

		nestedEntries, err := GetEntries(nestedFe, nil)
		require.NoError(t, err)
		assert.Len(t, nestedEntries, 1)
		assert.Contains(t, nestedEntries[0].Name, "nested.tar#inner.txt")
		assert.Equal(t, []byte("inner content"), nestedEntries[0].Content.Data)
	})

	t.Run("non-container file", func(t *testing.T) {
		filePath := filepath.Join(projectRoot, "testFiles", "data", "data1.yaml")
		fe := NewFileEntry(filePath)
		_, err := GetEntries(fe, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not a container")
	})
}
