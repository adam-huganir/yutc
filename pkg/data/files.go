package data

import (
	"errors"

	"github.com/adam-huganir/yutc/pkg/loader"
	"github.com/rs/zerolog"
)

// CountDataRecursables counts the number of recursable (directory or archive) items in the Input list.
func CountDataRecursables(paths []*Input) (int, error) {
	recursables := 0
	for _, f := range paths {
		isContainer, err := f.IsContainer()
		if err != nil {
			return recursables, err
		}
		if isContainer {
			recursables++
		}
	}
	return recursables, nil
}

// ResolveDataPaths parses data path strings, loads their content, and expands directories.
func ResolveDataPaths(paths []string, tempDir string, logger *zerolog.Logger) ([]*Input, error) {
	var outFiles []*Input
	for _, p := range paths {
		dis, err := ParseDataArgWithTempDir(p, tempDir)
		if err != nil {
			return nil, err
		}
		for _, di := range dis {
			di.SetLogger(logger)
			err = di.Load()
			if err != nil && !errors.Is(err, loader.ErrIsContainer) {
				return nil, err
			} else if err != nil {
				// For data, expand the directory into child Inputs
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

// expandDataContainer walks a directory or archive and creates DataInput entries for each file found.
func expandDataContainer(di *Input, outFiles *[]*Input, logger *zerolog.Logger) error {
	*outFiles = append(*outFiles, di) // include the directory itself (skipped during merge)
	entries, err := loader.GetEntries(di.FileEntry, logger)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		isDir, err := entry.IsDir()
		if err != nil {
			return err
		}
		if isDir {
			continue
		}
		child := &Input{
			FileEntry: entry,
		}
		err = child.Load()
		if err != nil {
			return err
		}
		*outFiles = append(*outFiles, child)
	}
	return nil
}
