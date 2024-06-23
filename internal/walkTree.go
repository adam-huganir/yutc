package internal

import (
	"fmt"
	"github.com/spf13/afero"
	"io/fs"
	"path"
	"path/filepath"
)

func WalkDir(rootPath string) []string {
	var files []string
	var err error
	isDir := false
	YutcLog.Trace().Msg(fmt.Sprintf("WalkDir(%s, %s)", rootPath, Fs))

	isArchive := IsArchive(rootPath)
	if !isArchive {
		isDir, err = IsDir(rootPath)
	}
	if err != nil {
		YutcLog.Fatal().Msg(err.Error())
	} else if !(isArchive || isDir) {
		YutcLog.Fatal().Msg(fmt.Sprintf("%s is not a recursive directory/archive", rootPath))
	}
	if isDir {
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
	} else if isArchive {
		YutcLog.Fatal().Msg("Archive files are not supported yet. Please provide a recursive directory. Exiting... ")
	} else {
		YutcLog.Fatal().Msg(fmt.Sprintf("%s is not a recursive directory/archive", rootPath))
	}

	return files
}

func NormalizeFilepath(file string) string {
	return filepath.ToSlash(filepath.Clean(path.Join(file)))
}
