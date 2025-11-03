package internal

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/spf13/afero"
)

func WalkDir(rootPath string) []string {
	var files []string
	YutcLog.Trace().Msg(fmt.Sprintf("WalkDir(%s, %s)", rootPath, Fs))

	isDir, err := afero.IsDir(Fs, rootPath)
	if !isDir || err != nil {
		panic(fmt.Sprintf("%s is not a directory", rootPath))
	}
	err = afero.Walk(Fs, rootPath,
		func(path string, info fs.FileInfo, err error) error {
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

func NormalizeFilepath(file string) string {
	return filepath.ToSlash(filepath.Clean(path.Join(file)))
}
