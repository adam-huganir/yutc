package internal

import (
	"bytes"
	"errors"
	"github.com/spf13/afero"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var Fs = afero.NewOsFs()

// GetDataFromPath reads from a file, URL, or stdin and returns a buffer with the contents
func GetDataFromPath(source, arg string, settings *YutcSettings) (*bytes.Buffer, error) {
	var err error
	buff := new(bytes.Buffer)
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
		buff, err = getUrlFile(arg, buff, settings)
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
func getUrlFile(arg string, buff *bytes.Buffer, settings *YutcSettings) (*bytes.Buffer, error) {
	var header http.Header
	if settings.BearerToken != "" {
		header = http.Header{
			"Authorization": []string{"Bearer " + settings.BearerToken},
		}
	}
	urlParsed, err := url.Parse(arg)
	if err != nil {
		return nil, err

	}
	if settings.BasicAuth != "" {
		username := strings.SplitN(settings.BearerToken, ":", 2)
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

// Exists checks if a path exists, returns a bool pointer and an error if doesn't exist
func Exists(path string) (bool, error) {
	var exists bool
	exists, err := afero.Exists(Fs, path)
	if err != nil {
		return exists, err
	}
	return exists, nil
}

// GenerateTempDirName generates a temporary directory name, basically just standard's MktempDir's without the create
func GenerateTempDirName(pattern string) (string, error) {
	// stole this from standard lib MktempDir's gen
	prefix, suffix := "", ""
	for i := 0; i < len(pattern); i++ {
		if os.IsPathSeparator(pattern[i]) {
			return "", errors.New("pattern contains path separator")
		}
	}
	if pos := strings.LastIndexByte(pattern, '*'); pos != -1 {
		prefix, suffix = pattern[:pos], pattern[pos+1:]
	} else {
		prefix = pattern
	}
	try := 0
	for {
		name := prefix + strconv.Itoa(rand.Intn(100000000)) + suffix
		_, err := os.Stat(name)
		if os.IsNotExist(err) {
			return name, nil
		} else {
			if try++; try < 10000 {
				continue
			}
			return "", &os.PathError{Op: "createtemp", Path: prefix + "*" + suffix, Err: os.ErrExist}
		}
	}
}

// CheckIfDir checks if a path is a directory, returns a bool pointer and an error if doesn't exist
func CheckIfDir(path string) (bool, error) {
	var isDir bool
	isDir, err := afero.IsDir(Fs, path)
	if err != nil {
		return isDir, err
	}
	return isDir, nil
}

// CheckIfFile checks if a path is a file, returns a bool pointer and an error if doesn't exist
func CheckIfFile(path string) (bool, error) {
	var isFile bool
	fileInfo, err := Fs.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			isFile = true
			return isFile, err
		}
		return false, err
	}
	return !fileInfo.IsDir(), nil
}

func CountRecursables(paths []string) (int, error) {
	recursables := 0
	for _, templatePath := range paths {
		source, err := ParseFileStringFlag(templatePath)
		if source != "file" {
			if source == "url" {
				if IsArchive(templatePath) {
					recursables++
				}
			}
			continue
		}
		isDir, err := CheckIfDir(templatePath)
		if err != nil {
			return recursables, err
		} else if isDir || IsArchive(templatePath) {
			recursables++
		}
	}
	return recursables, nil
}
