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
	"text/template"

	"github.com/rs/zerolog"
)

func TemplateFilenames(fas []*FileArg, t *template.Template, data map[string]any) error {
	for _, fa := range fas {
		_, err := fa.TemplateName(t, data)
		if err != nil {
			return err
		}
	}
	return nil
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
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
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
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// CheckIfFile checks if a path is a file, returns a bool pointer and an error if doesn't exist
func CheckIfFile(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, err
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
				if IsArchive(f.Name) {
					recursables++
				}
			}
			continue
		}
		isDir, err := IsDir(f.Name)
		if err != nil {
			return recursables, err
		} else if isDir || IsArchive(f.Name) {
			recursables++
		}
	}
	return recursables, nil
}

// ResolvePaths introspects each path and resolves it to actual file paths.
// If a path is a directory, it resolves all data in that directory.
// After applying any match/exclude patterns, returns the list of data.
func ResolvePaths(paths []string, kind FileKind, tempDir string, logger *zerolog.Logger) (outFiles []*FileArg, err error) {
	//fileArgs := make([]*FileArg, 0, len(paths))
	for _, p := range paths {
		fas, err := ParseFileArg(p, kind)
		if err != nil {
			return nil, err
		}
		for _, f := range fas {
			f.SetLogger(logger)
			err = f.Load()
			if err != nil && !strings.HasSuffix(err.Error(), " is a container") {
				return nil, err
			} else if err != nil {
				err = f.LoadContainer()
				if err != nil {
					return nil, err
				}
			}
			outFiles = append(outFiles, f)
		}
	}

	return outFiles, nil
}
func resolvePath(path string, kind *FileKind, tempDir string, logger *zerolog.Logger) (outFiles []*FileArg, err error) {
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
	//			recursedFiles, _ := WalkDir(f.Name, logger)
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
	//			if f.Name != "-" {
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
	//
	//logger.Debug().Msgf("Found %d data", len(outFiles))
	//for _, commonFile := range outFiles {
	//	var urlRepr string
	//	if commonFile.Url != nil {
	//		urlRepr = commonFile.Url.String()
	//	}
	//
	//	logger.Trace().Msgf("  - %s (%s from %s) %s", commonFile.Name, commonFile.Kind, commonFile.Source, urlRepr)
	//}
	return outFiles, nil
}
