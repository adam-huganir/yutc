package internal

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"slices"
)

func WalkDir(rootPath string, p fs.FS, match []string) []string {
	YutcLog.Trace().Msg(fmt.Sprintf("WalkDir(%s, %s, %s)", rootPath, p, match))

	// for windows:

	var files []string
	_ = fs.WalkDir(p, ".",
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			files = append(files, path)
			return nil
		},
	)

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

	for i, file := range output {
		output[i] = filepath.ToSlash(path.Join(rootPath, file))
	}
	return output
}
