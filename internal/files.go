package internal

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
)

func GetDataFromPath(path string) (*bytes.Buffer, error) {
	var err error
	url, err := ParseFileStringFlag(path)
	if err != nil {
		return nil, err
	}
	if url.Scheme == "file" {
		var stat os.FileInfo
		if stat, err = os.Stat(url.Path); err != nil {
			if os.IsNotExist(err) {
				return nil, errors.New("file does not exist: " + url.Path)
			} else {
				return nil, err
			}
		}
		if stat.IsDir() {
			return nil, errors.New("path is a directory: " + url.Path)
		}
		contents, err := os.ReadFile(url.Path)
		if err != nil {
			return nil, err
		}
		return bytes.NewBuffer(contents), nil
	}
	if url.Scheme == "http" || url.Scheme == "https" {
		response, err := http.Get(url.String())
		if err != nil {
			return nil, err
		}
		return GetDataFromFile(response.Body)
	}
	// TODO: care more about how we fail here
	return nil, errors.New("unsupported scheme, " + url.Scheme + ", for url: " + url.String())
}

func GetDataFromFile(f io.ReadCloser) (*bytes.Buffer, error) {
	var err error
	var contents []byte
	defer func() { _ = f.Close() }()
	if contents, err = io.ReadAll(f); err != nil {
		return bytes.NewBuffer(contents), nil
	}
	return nil, err
}
