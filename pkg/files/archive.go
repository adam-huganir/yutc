// Package files provides file system operations, archive handling, and URL fetching capabilities.
package files

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path"
	"slices"
	"strings"
)

// FilePathMap represents a file extracted from an archive with its path and data.
type FilePathMap struct {
	FilePath string
	Data     []byte
}

// IsArchive checks if a file path has an archive extension (.tgz, .tar.gz, .tar, .zip, .gz).
func IsArchive(filePath string) bool {
	for _, suffix := range []string{".tgz", ".tar.gz", ".tar", ".zip", ".gz"} {
		if strings.ToLower(path.Ext(filePath)) == suffix {
			return true
		}
	}
	return false
}

// ReadTar extracts all files from a tar or tar.gz archive and returns them as FilePathMap entries.
func ReadTar(filePath string) ([]FilePathMap, error) {
	var tarReader *tar.Reader
	var header *tar.Header
	var err error
	var files []FilePathMap
	var f *os.File
	var gz *gzip.Reader

	f, err = os.Open(filePath)
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)
	if err != nil {
		return nil, err
	}
	if slices.Contains([]string{".tgz", ".tar.gz"}, path.Ext(filePath)) {
		gz, err = gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		tarReader = tar.NewReader(gz)
	} else {
		tarReader = tar.NewReader(f)
	}
	for {
		header, err = tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		fpm := FilePathMap{
			FilePath: header.Name,
			Data:     make([]byte, header.Size),
		}
		_, err = io.ReadFull(tarReader, fpm.Data)
		if err != nil {
			return nil, err
		}
		println(fpm.FilePath)
		println(fpm.Data[:100])
		files = append(files, fpm)
	}
	return files, nil
}
