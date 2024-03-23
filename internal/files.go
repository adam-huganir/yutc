package internal

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// IsDir Simple helper function to check if a path is a directory
func IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

// GetDataFromPath reads from a file, URL, or stdin and returns a buffer with the contents
func GetDataFromPath(source, arg string) (*bytes.Buffer, error) {
	var err error
	buff := new(bytes.Buffer)
	if err != nil {
		return nil, err
	}
	switch source {
	case "file":
		var stat os.FileInfo
		if stat, err = os.Stat(arg); err != nil {
			if os.IsNotExist(err) {
				return nil, errors.New("file does not exist: " + arg)
			} else {
				return nil, err
			}
		}
		if stat.IsDir() {
			return nil, errors.New("path is a directory: " + arg)
		}
		contents, err := os.ReadFile(arg)
		buff = bytes.NewBuffer(contents)
		if err != nil {
			return nil, err
		}
	case "url":
		buff, err = getUrlFile(arg, buff)
		if err != nil {
			return nil, errors.New("error reading from url: " + arg)
		}
	case "stdin":
		buff, err = GetDataFromReadCloser(os.Stdin)
		if err != nil {
			return nil, errors.New("error reading from stdin")
		}
	default:
		return nil, errors.New("unsupported scheme/source for input: " + arg)
	}
	if buff == nil {
		return nil, errors.New("unknown error reading from source: " + arg)
	}
	return buff, nil
}

// getUrlFile reads a file from a URL and returns a buffer with the contents, auth optional based on config
func getUrlFile(arg string, buff *bytes.Buffer) (*bytes.Buffer, error) {
	var header http.Header

	if RunSettings.BearerToken != "" {
		header = http.Header{
			"Authorization": []string{"Bearer " + RunSettings.BearerToken},
		}
	}
	urlParsed, err := url.Parse(arg)
	if err != nil {
		return nil, err

	}
	if RunSettings.BasicAuth != "" {
		username := strings.SplitN(RunSettings.BearerToken, ":", 2)
		user := url.UserPassword(username[0], username[1])
		urlParsed.User = user
	}
	req := http.Request{
		Method: "GET",
		URL:    urlParsed,
		Header: header,
	}
	response, err := http.DefaultClient.Do(&req)
	if err != nil {
		return nil, err
	}
	buff, err = GetDataFromReadCloser(response.Body)
	if err != nil {
		return nil, err
	}
	return buff, nil
}

// GetDataFromReadCloser reads from a ReadCloser and returns a buffer with the contents
func GetDataFromReadCloser(f io.ReadCloser) (*bytes.Buffer, error) {
	var err error
	var contents []byte
	//defer func() { _ = f.Close() }()
	if contents, err = io.ReadAll(f); err == nil {
		return bytes.NewBuffer(contents), nil
	}
	return nil, err
}
