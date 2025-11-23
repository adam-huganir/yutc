package files

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/afero"
)

// Fs is the global filesystem abstraction used throughout the files package.
var Fs = initFs(afero.NewOsFs)

func initFs(fsCreator func() afero.Fs) afero.Fs {
	return fsCreator()
}

// GetDataFromPath reads from a file, URL, or stdin and returns a buffer with the contents
func GetDataFromPath(source, arg, bearerToken, basicAuth string) (*bytes.Buffer, error) {
	var err error
	var buff *bytes.Buffer
	switch source {
	case "file":
		var stat os.FileInfo
		if stat, err = os.Stat(arg); err != nil {
			if os.IsNotExist(err) {
				return nil, errors.New("file does not exist: " + arg)
			}
			return nil, err
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
		buff, err = getURLFile(arg, bearerToken, basicAuth)
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

// getURLFile reads a file from a URL and returns a buffer with the contents, auth optional based on config
func getURLFile(arg, bearerToken, basicAuth string) (*bytes.Buffer, error) {
	var header http.Header
	if bearerToken != "" {
		header = http.Header{
			"Authorization": []string{"Bearer " + bearerToken},
		}
	}
	urlParsed, err := url.Parse(arg)
	if err != nil {
		return nil, err

	}
	if basicAuth != "" {
		username := strings.SplitN(basicAuth, ":", 2)
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
	buff, err := GetDataFromReadCloser(response.Body)
	if err != nil {
		return nil, err
	}
	return buff, nil
}

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
func CountRecursables(paths []string) (int, error) {
	recursables := 0
	for _, templatePath := range paths {
		source, err := ParseFileStringFlag(templatePath)
		if err != nil {
			return recursables, err
		}
		if source != "file" {
			if source == "url" {
				if IsArchive(templatePath) {
					recursables++
				}
			}
			continue
		}
		isDir, err := IsDir(templatePath)
		if err != nil {
			return recursables, err
		} else if isDir || IsArchive(templatePath) {
			recursables++
		}
	}
	return recursables, nil
}

// ParseFileStringFlag determines the source of a file string flag based on format and returns the source
// as a string, or an error if the source is not supported. Currently, supports "file", "url", and "stdin" (as `-`).
func ParseFileStringFlag(v string) (string, error) {
	if !strings.Contains(v, "://") {
		if v == "-" {
			return "stdin", nil
		}
		_, err := filepath.Abs(v)
		if err != nil {
			return "", err
		}
		return "file", nil
	}
	if v == "-" {
		return "stdin", nil
	}
	allowedURLPrefixes := []string{"http://", "https://"}
	for _, prefix := range allowedURLPrefixes {
		if strings.HasPrefix(v, prefix) {
			return "url", nil
		}
	}
	return "", errors.New("unsupported scheme/source for input: " + v)
}

// ResolvePaths introspects each path and resolves it to actual file paths.
// If a path is a directory, it resolves all files in that directory.
// After applying any match/exclude patterns, returns the list of files.
func ResolvePaths(paths []string, tempDir string, logger *zerolog.Logger) ([]string, error) {
	var outFiles []string
	var filename string
	var data []byte
	recursables, err := CountRecursables(paths)
	if err != nil {
		return nil, err
	}

	if recursables > 0 {
		for _, templatePath := range paths {
			source, err := ParseFileStringFlag(templatePath)
			if err != nil {
				return nil, err
			}
			switch source {
			case "stdin":
			case "url":
				filename, data, _, err = ReadURL(templatePath, logger)
				tempPath := filepath.Join(tempDir, filename)
				if err != nil {
					return nil, err
				}
				tempDirExists, err := Exists(tempPath)
				if err != nil {
					return nil, err
				}
				if !tempDirExists {
					err = os.Mkdir(tempPath, 0o755)
					if err != nil {
						logger.Panic().Msg(err.Error())
					}
				}
				err = os.WriteFile(tempPath, data, 0o644)
				if err != nil {
					return nil, err
				}
				templatePath = tempPath
				fallthrough
			default:
				templatePath = filepath.ToSlash(templatePath)
				filteredPaths := WalkDir(templatePath, logger)
				outFiles = append(outFiles, filteredPaths...)
			}
		}
	} else {
		for _, templatePath := range paths {
			source, err := ParseFileStringFlag(templatePath)
			if err != nil {
				panic(err)
			}
			if source == "url" {
				filename, data, _, err := ReadURL(templatePath, logger)
				tempPath := filepath.Join(tempDir, filename)
				if err != nil {
					logger.Fatal().Msg(err.Error())
				}
				errRaw := os.WriteFile(tempPath, data, 0o644)
				if errRaw != nil {
					return nil, errRaw
				}
				templatePath = tempPath
			}
			outFiles = append(outFiles, templatePath)
		}
	}

	logger.Debug().Msgf("Found %d common template files", len(outFiles))
	for _, commonFile := range outFiles {
		logger.Trace().Msg("  - " + commonFile)
	}
	return outFiles, nil
}

// CountDataRecursables counts the number of recursable (directory or archive) data files
func CountDataRecursables(dataFiles []string) (int, error) {
	recursables := 0
	for _, dataFileArg := range dataFiles {
		dataArg, err := ParseDataFileArg(dataFileArg)
		if err != nil {
			return recursables, err
		}

		source, err := ParseFileStringFlag(dataArg.Path)
		if err != nil {
			return recursables, err
		}
		if source != "file" {
			if source == "url" {
				if IsArchive(dataArg.Path) {
					recursables++
				}
			}
			continue
		}
		isDir, err := IsDir(dataArg.Path)
		if err != nil {
			return recursables, err
		} else if isDir || IsArchive(dataArg.Path) {
			recursables++
		}
	}
	return recursables, nil
}
