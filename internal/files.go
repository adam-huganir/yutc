package internal

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
)

func GetFile(url *url.URL) (*bytes.Buffer, error) {
	var err error
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
		contents, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		return bytes.NewBuffer(contents), nil
	}
	// TODO: care more about how we fail here
	return nil, errors.New("unsupported scheme, " + url.Scheme + ", for url: " + url.String())
}
