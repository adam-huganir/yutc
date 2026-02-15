package data

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

// GetDataFromReadCloser reads from a ReadCloser and returns a buffer with the contents
func GetDataFromReadCloser(f io.ReadCloser) (*bytes.Buffer, error) {
	var err error
	var contents []byte
	if contents, err = io.ReadAll(f); err == nil {
		return bytes.NewBuffer(contents), nil
	}
	return nil, err
}

// Exists checks if a path exists, returns a bool pointer and an error if it doesn't exist
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
		someRand := strconv.Itoa(rand.Intn(100000000))
		name := filepath.Join(os.TempDir(), prefix+someRand+suffix)
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

// IsFile checks if a path is a file, returns a bool pointer and an error if doesn't exist
func IsFile(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, err
		}
		return false, err
	}
	return !fileInfo.IsDir(), nil
}

// CountDataRecursables counts the number of recursable (directory or archive) items in the DataInput list.
func CountDataRecursables(paths []*DataInput) (int, error) {
	recursables := 0
	for _, f := range paths {
		if f.Source != SourceKindFile {
			if f.Source == SourceKindURL {
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

// CountTemplateRecursables counts the number of recursable (directory or archive) items in the TemplateInput list.
func CountTemplateRecursables(paths []*TemplateInput) (int, error) {
	recursables := 0
	for _, f := range paths {
		if f.Source != SourceKindFile {
			if f.Source == SourceKindURL {
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

// ResolveDataPaths parses data path strings, loads their content, and expands directories.
func ResolveDataPaths(paths []string, logger *zerolog.Logger) ([]*DataInput, error) {
	var outFiles []*DataInput
	for _, p := range paths {
		dis, err := ParseDataArg(p)
		if err != nil {
			return nil, err
		}
		for _, di := range dis {
			di.SetLogger(logger)
			err = di.Load()
			if err != nil && !errors.Is(err, ErrIsContainer) {
				return nil, err
			} else if err != nil {
				// For data, expand the directory into child DataInputs
				err = expandDataContainer(di, &outFiles, logger)
				if err != nil {
					return nil, err
				}
				continue
			}
			outFiles = append(outFiles, di)
		}
	}
	return outFiles, nil
}

// expandDataContainer walks a directory and creates DataInput entries for each file found.
func expandDataContainer(di *DataInput, outFiles *[]*DataInput, logger *zerolog.Logger) error {
	*outFiles = append(*outFiles, di) // include the directory itself (skipped during merge)
	paths, err := WalkDir(di.FileEntry, logger)
	if err != nil {
		return err
	}
	for _, p := range paths {
		if di.Name == p {
			continue
		}
		isDir, err := IsDir(p)
		if err != nil {
			return err
		}
		if isDir {
			continue
		}
		child := NewDataInput(p, []FileEntryOption{WithSource(SourceKindFile)})
		child.SetLogger(logger)
		err = child.Load()
		if err != nil {
			return err
		}
		*outFiles = append(*outFiles, child)
	}
	return nil
}
