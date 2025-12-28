package data

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/afero"
)

// Fs is the global filesystem abstraction used throughout the data package.
var Fs = initFs(afero.NewOsFs)

func initFs(fsCreator func() afero.Fs) afero.Fs {
	return fsCreator()
}

// GetDataFromPath reads from a file, URL, or stdin and returns a buffer with the contents
//func GetDataFromPath(source, arg, bearerToken, basicAuth string) (*bytes.Buffer, error) {
//	var err error
//	var buff *bytes.Buffer
//	switch source {
//	case "file":
//		var stat os.FileInfo
//		if stat, err = os.Stat(arg); err != nil {
//			if os.IsNotExist(err) {
//				return nil, errors.New("file does not exist: " + arg)
//			}
//			return nil, err
//		}
//		if stat.IsDir() {
//			return nil, errors.New("path is a directory: " + arg)
//		}
//		contents, err := os.ReadFile(arg)
//		buff = bytes.NewBuffer(contents)
//		if err != nil {
//			return nil, err
//		}
//	case "url":
//		buff, err = getURLFile(arg, bearerToken, basicAuth)
//		if err != nil {
//			return nil, errors.New("error reading from url: " + arg)
//		}
//	case "stdin":
//		buff, err = GetDataFromReadCloser(os.Stdin)
//		if err != nil {
//			return nil, errors.New("error reading from stdin")
//		}
//	default:
//		return nil, errors.New("unsupported scheme/source for input: " + arg)
//	}
//	if buff == nil {
//		return nil, errors.New("unknown error reading from source: " + arg)
//	}
//	return buff, nil
//}
//
//// getURLFile reads a file from a URL and returns a buffer with the contents, auth optional based on config
//func getURLFile(arg, bearerToken, basicAuth string) (*bytes.Buffer, error) {
//	var header http.Header
//	if bearerToken != "" {
//		header = http.Header{
//			"Authorization": []string{"Bearer " + bearerToken},
//		}
//	}
//	urlParsed, err := url.Parse(arg)
//	if err != nil {
//		return nil, err
//
//	}
//	if basicAuth != "" {
//		username := strings.SplitN(basicAuth, ":", 2)
//		user := url.UserPassword(username[0], username[1])
//		urlParsed.User = user
//	}
//	req := http.Request{
//		Method: "GET",
//		URL:    urlParsed,
//		Header: header,
//	}
//	response, err := http.DefaultClient.Do(&req)
//	if err != nil {
//		return nil, err
//	}
//	buff, err := GetDataFromReadCloser(response.Body)
//	if err != nil {
//		return nil, err
//	}
//	return buff, nil
//}

// GetDataFromReadCloser reads from a ReadCloser and returns a buffer with the contents
func GetDataFromReadCloser(f io.ReadCloser) (*bytes.Buffer, error) {
	var err error
	var contents []byte
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

func MakeDirExist(path string) (err error) {
	dirExists, err := Exists(path)
	if err != nil {
		return fmt.Errorf("unable to check directory %s exists: %w", path, err)
	}
	if !dirExists {
		err = os.Mkdir(path, 0o755)
		if err != nil {
			return fmt.Errorf("unable to create directory %s: %w", path, err)
		}
	}
	return nil
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
		try++
		if os.IsNotExist(err) {
			return name, nil
		} else if try < 10000 {
			continue
		}
		return "", &os.PathError{Op: "createtemp", Path: prefix + "*" + suffix, Err: os.ErrExist}
	}
}

// IsDir checks if a path is a directory, returns a bool pointer and an error if doesn't exist
func IsDir(path string) (bool, error) {
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

// CountRecursables counts the number of recursable (directory or archive) items in the path list.
func CountRecursables(paths []*FileArg) (int, error) {
	recursables := 0
	for _, f := range paths {
		if f.Source != "file" {
			if f.Source == "url" {
				if IsArchive(f.Path) {
					recursables++
				}
			}
			continue
		}
		isDir, err := IsDir(f.Path)
		if err != nil {
			return recursables, err
		} else if isDir || IsArchive(f.Path) {
			recursables++
		}
	}
	return recursables, nil
}

// ResolvePaths introspects each path and resolves it to actual file paths.
// If a path is a directory, it resolves all data in that directory.
// After applying any match/exclude patterns, returns the list of data.
func ResolvePaths(paths []string, kind string, tempDir string, logger *zerolog.Logger) (outFiles []*FileArg, err error) {
	//fileArgs := make([]*FileArg, 0, len(paths))
	for _, p := range paths {
		f, err := ParseFileArg(p, kind)
		if err != nil {
			return nil, err
		}
		f.SetLogger(logger)
		err = f.Load()
		if err != nil {
			return nil, err
		}
		outFiles = append(outFiles, f)
	}
	//
	//recursables, err := CountRecursables(fileArgs)
	//if err != nil {
	//	return nil, err
	//}
	//
	//if recursables > 0 {
	//	for _, f := range fileArgs {
	//		switch f.Source {
	//		case "stdin":
	//			err = f.Load(logger)
	//			if err != nil {
	//				return outFiles, err
	//			}
	//			outFiles = append(outFiles, f)
	//		case "url":
	//
	//			if err != nil {
	//				return outFiles, err
	//			}
	//		default:
	//			recursedFiles := WalkDir(f.Path, logger)
	//			recursedFileArgs := make([]*FileArg, len(recursedFiles))
	//			for i, fp := range recursedFiles {
	//				recursedFileArgs[i], err = ParseFileArg(fp, kind)
	//				if err != nil {
	//					return nil, err
	//				}
	//			}
	//
	//			outFiles = append(outFiles, recursedFileArgs...)
	//		}
	//	}
	//} else {
	//	for _, f := range fileArgs {
	//		switch f.Source {
	//		case "stdin":
	//			buf, err := GetDataFromReadCloser(os.Stdin)
	//			if err != nil {
	//				return outFiles, err
	//			}
	//			f.Content.Data = buf.Bytes()
	//			if f.Path != "-" {
	//				panic("a bug yo")
	//			}
	//		case "url":
	//			err = urlToFile(f, tempDir, logger)
	//			if err != nil {
	//				return nil, err
	//			}
	//		}
	//		outFiles = append(outFiles, f)
	//	}
	//}

	logger.Debug().Msgf("Found %d data", len(outFiles))
	for _, commonFile := range outFiles {
		var urlRepr string
		if commonFile.Url != nil {
			urlRepr = commonFile.Url.String()
		}

		logger.Trace().Msgf("  - %s (%s from %s) %s", commonFile.Path, commonFile.Kind, commonFile.Source, urlRepr)
	}
	return outFiles, nil
}
