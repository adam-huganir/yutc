package internal

import (
	"fmt"
	"github.com/spf13/afero"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"slices"
)

func WalkDir(rootPath string, match []string) []string {
	var files []string
	YutcLog.Trace().Msg(fmt.Sprintf("WalkDir(%s, %s, %s)", rootPath, Fs, match))

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

	var output []string
	if len(match) > 0 {
		for _, pattern := range match {
			var matcher *regexp.Regexp
			not := pattern[0] == '!'
			if not {
				matcher = regexp.MustCompile(pattern[1:])
			} else {
				matcher = regexp.MustCompile(pattern)
			}
			for _, file := range files {
				if !not && matcher.MatchString(file) && !slices.Contains(output, file) {
					output = append(output, file)
				} else if not && !matcher.MatchString(file) && !slices.Contains(output, file) {
					output = append(output, file)
				}
			}
		}
		YutcLog.Trace().Msg(fmt.Sprintf("%d files matched include patterns", len(output)))
	} else {
		output = files
		YutcLog.Trace().Msg(fmt.Sprintf("No patterns provided, %d paths passed through", len(output)))
	}

	return output
}

func NormalizeFilepath(file string) string {
	return filepath.ToSlash(filepath.Clean(path.Join(file)))
}
