package internal

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path"
	"slices"
)

type filePathMap struct {
	FilePath string
	Data     []byte
}

func IsArchive(filePath string) bool {
	return slices.Contains([]string{".tgz", ".tar.gz", ".tar"}, path.Ext(filePath))
}

func ReadTar(filePath string) ([]filePathMap, error) {
	var tarReader *tar.Reader
	var header *tar.Header
	var err error
	var files []filePathMap
	var f *os.File
	var gz *gzip.Reader

	f, err = os.Open(filePath)
	defer f.Close()
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
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		fpm := filePathMap{
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
