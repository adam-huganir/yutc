package loader

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path"
	"strings"
)

// FilePathMap represents a file extracted from an archive with its path and data.
type FilePathMap struct {
	FilePath string
	Data     []byte
}

// IsArchive checks if a file path has an archive extension (.tgz, .tar.gz, .tar, .zip, .gz).
func IsArchive(filePath string) bool {
	lowerPath := strings.ToLower(filePath)
	for _, suffix := range []string{".tgz", ".tar.gz", ".tar", ".zip", ".gz"} {
		if strings.HasSuffix(lowerPath, suffix) {
			return true
		}
	}
	return false
}

// ReadArchive dispatches to the correct reader based on file extension.
func ReadArchive(filePath string) ([]FilePathMap, error) {
	lowerPath := strings.ToLower(filePath)
	if strings.HasSuffix(lowerPath, ".zip") {
		return ReadZip(filePath)
	}
	if strings.HasSuffix(lowerPath, ".gz") && !strings.HasSuffix(lowerPath, ".tar.gz") && !strings.HasSuffix(lowerPath, ".tgz") {
		return ReadGzip(filePath)
	}
	// Default to Tar (handles .tar, .tar.gz, .tgz)
	return ReadTar(filePath)
}

// ReadArchiveFromBytes dispatches to the correct reader based on name and data.
func ReadArchiveFromBytes(name string, data []byte) ([]FilePathMap, error) {
	lowerPath := strings.ToLower(name)
	if strings.HasSuffix(lowerPath, ".zip") {
		return ReadZipFromBytes(data)
	}
	if strings.HasSuffix(lowerPath, ".gz") && !strings.HasSuffix(lowerPath, ".tar.gz") && !strings.HasSuffix(lowerPath, ".tgz") {
		return ReadGzipFromBytes(name, data)
	}
	// Default to Tar (handles .tar, .tar.gz, .tgz)
	return ReadTarFromBytes(name, data)
}

// ReadTar extracts all data from a tar or tar.gz archive and returns them as FilePathMap entries.
func ReadTar(filePath string) ([]FilePathMap, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadTarFromReader(filePath, f)
}

// ReadTarFromBytes extracts all data from a tar or tar.gz archive in memory.
func ReadTarFromBytes(name string, data []byte) ([]FilePathMap, error) {
	return ReadTarFromReader(name, bytes.NewReader(data))
}

// ReadTarFromReader extracts all data from a tar or tar.gz archive from a reader.
func ReadTarFromReader(name string, r io.Reader) ([]FilePathMap, error) {
	var tarReader *tar.Reader
	var header *tar.Header
	var err error
	var files []FilePathMap
	var gz *gzip.Reader

	if strings.HasSuffix(strings.ToLower(name), ".tgz") || strings.HasSuffix(strings.ToLower(name), ".tar.gz") {
		gz, err = gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		tarReader = tar.NewReader(gz)
	} else {
		tarReader = tar.NewReader(r)
	}
	for {
		header, err = tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		// Only process regular files
		if header.Typeflag != tar.TypeReg {
			continue
		}

		data, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, err
		}

		files = append(files, FilePathMap{
			FilePath: header.Name,
			Data:     data,
		})
	}
	return files, nil
}

// ReadZip extracts all data from a zip archive and returns them as FilePathMap entries.
func ReadZip(filePath string) ([]FilePathMap, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var files []FilePathMap
	for _, f := range r.File {
		// Skip directories
		if f.FileInfo().IsDir() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}

		files = append(files, FilePathMap{
			FilePath: f.Name,
			Data:     data,
		})
	}
	return files, nil
}

// ReadZipFromBytes extracts all data from a zip archive in memory.
func ReadZipFromBytes(data []byte) ([]FilePathMap, error) {
	reader := bytes.NewReader(data)
	r, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return nil, err
	}

	var files []FilePathMap
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}

		files = append(files, FilePathMap{
			FilePath: f.Name,
			Data:     data,
		})
	}
	return files, nil
}

// ReadGzip reads a single gzipped file and returns it as a FilePathMap.
func ReadGzip(filePath string) ([]FilePathMap, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadGzipFromReader(filePath, f)
}

// ReadGzipFromBytes reads a single gzipped file from memory.
func ReadGzipFromBytes(name string, data []byte) ([]FilePathMap, error) {
	return ReadGzipFromReader(name, bytes.NewReader(data))
}

// ReadGzipFromReader reads a single gzipped file from a reader.
func ReadGzipFromReader(name string, r io.Reader) ([]FilePathMap, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	data, err := io.ReadAll(gz)
	if err != nil {
		return nil, err
	}

	// Determine original filename if possible, otherwise use base name without .gz
	outName := gz.Name
	if outName == "" {
		outName = strings.TrimSuffix(path.Base(name), ".gz")
	}

	return []FilePathMap{
		{
			FilePath: outName,
			Data:     data,
		},
	}, nil
}
