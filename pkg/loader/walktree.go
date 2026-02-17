package loader

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/rs/zerolog"
)

// WalkDir recursively walks a directory and returns a list of all file paths.
func WalkDir(root *FileEntry, logger *zerolog.Logger) (files []string, err error) {
	if logger != nil {
		logger.Trace().Msg(fmt.Sprintf("WalkDir(%s)", root.Name))
	}
	if root == nil {
		return nil, fmt.Errorf("root is nil")
	}

	isDir, err := IsDir(root.Name)
	if !isDir || err != nil {
		return nil, fmt.Errorf("this code branch was empty, " +
			"so whenever we run into this we should figure out what was supposed to go here")
	}
	err = filepath.WalkDir(root.Name,
		func(path string, _ fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			files = append(files, NormalizeFilepath(path))
			return nil
		},
	)
	if err != nil {
		return files, fmt.Errorf("error walking directory %s: %w", root.Name, err)
	}
	return files, err
}
