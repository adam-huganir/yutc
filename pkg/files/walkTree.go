package files

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/spf13/afero"
)

// WalkDir recursively walks a directory and returns a list of all file paths.
func WalkDir(rootPath string, logger *zerolog.Logger) []string {
	var files []string
	if logger != nil {
		logger.Trace().Msg(fmt.Sprintf("WalkDir(%s, %s)", rootPath, Fs))
	}

	isDir, err := afero.IsDir(Fs, rootPath)
	if !isDir || err != nil {
		panic("this code branch was empty, so whenever we run into this we should figure out what was supposed to go here")
	}
	err = afero.Walk(Fs, rootPath,
		func(path string, _ fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			files = append(files, NormalizeFilepath(path))
			return nil
		},
	)
	if err != nil {
		panic(fmt.Sprintf("Error walking directory %s: %s", rootPath, err))
	}
	return files
}

// NormalizeFilepath cleans and normalizes a file path to use forward slashes.
func NormalizeFilepath(file string) string {
	return filepath.ToSlash(filepath.Clean(path.Join(file)))
}
