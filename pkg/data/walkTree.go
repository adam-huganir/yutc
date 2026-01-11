package data

import (
	"fmt"
	"io/fs"

	"github.com/rs/zerolog"
	"github.com/spf13/afero"
)

// WalkDir recursively walks a directory and returns a list of all file paths.
func WalkDir(root *FileArg, logger *zerolog.Logger) (files []string, err error) {
	if logger != nil {
		logger.Trace().Msg(fmt.Sprintf("WalkDir(%s, %s)", root.Path, Fs))
	}
	if root == nil {
		return nil, fmt.Errorf("root is nil")
	}

	isDir, err := afero.IsDir(Fs, root.Path)
	if !isDir || err != nil {
		return nil, fmt.Errorf("this code branch was empty, " +
			"so whenever we run into this we should figure out what was supposed to go here")
	}
	err = afero.Walk(Fs, root.Path,
		func(path string, _ fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			files = append(files, NormalizeFilepath(path))
			return nil
		},
	)
	if err != nil {
		return files, fmt.Errorf("error walking directory %s: %w", root.Path, err)
	}
	return files, err
}
