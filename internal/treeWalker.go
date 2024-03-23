package internal

import (
	"fmt"
	"io/fs"
	"regexp"
	"slices"
)

func WalkDir(p fs.FS, match []string) []string {
	YutcLog.Trace().Msg(fmt.Sprintf("WalkDir(%s, %s)", p, match))

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

	if len(match) == 0 && len(match) == 0 {
		return files
	}

	var output []string
	if len(match) > 0 {
		for _, pattern := range match {
			matcher := regexp.MustCompile(pattern)
			for _, file := range files {
				if matcher.MatchString(file) && !slices.Contains(output, file) {
					output = append(output, file)
				}
			}
		}
		YutcLog.Trace().Msg(fmt.Sprintf("%d files matched include patterns", len(output)))

		// this lets us filter again by exclude even if include was set, although this currently considered an error
		// by the cli and currently not possible set both. However, we may want to add something later after
		// thinking about it more
		files = output
		output = []string{}
	}

	if len(match) > 0 {
		for _, pattern := range match {
			matcher := regexp.MustCompile(pattern)
			for _, file := range files {
				if !matcher.MatchString(file) {
					output = append(output, file)
				}
			}
		}
		YutcLog.Trace().Msg(fmt.Sprintf("%d files did not match matched exclude patterns", len(output)))
	}

	return output
}
