package loader

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/rs/zerolog"
)

// GetEntries returns a list of FileEntry objects for the contents of a container (directory or archive).
func GetEntries(root *FileEntry, logger *zerolog.Logger) (entries []*FileEntry, err error) {
	if logger != nil {
		logger.Trace().Msg(fmt.Sprintf("GetEntries(%s)", root.Name))
	}
	if root == nil {
		return nil, fmt.Errorf("root is nil")
	}

	isDir, err := root.IsDir()
	if err != nil {
		return nil, err
	}
	if isDir {
		type entryInfo struct {
			path  string
			isDir bool
		}
		var infos []entryInfo
		err = filepath.WalkDir(root.Name,
			func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				infos = append(infos, entryInfo{path: NormalizeFilepath(path), isDir: d.IsDir()})
				return nil
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error walking directory %s: %w", root.Name, err)
		}
		for _, info := range infos {
			if info.path == root.Name {
				continue
			}
			entries = append(entries, NewFileEntry(info.path,
				WithSource(SourceKindFile),
				WithLogger(logger),
				WithIsDir(info.isDir),
				WithIsFile(!info.isDir),
			))
		}
		return entries, nil
	}

	isArchive, err := root.IsArchive()
	if err != nil {
		return nil, err
	}
	if isArchive {
		var files []FilePathMap
		var err error
		if root.Content.Read {
			files, err = ReadArchiveFromBytes(root.Name, root.Content.Data)
		} else {
			files, err = ReadArchive(root.Name)
		}
		if err != nil {
			return nil, fmt.Errorf("error reading archive %s: %w", root.Name, err)
		}
		for _, f := range files {
			// Create a synthetic name that indicates it's inside an archive
			// This helps with debugging and potentially with resolution logic
			name := root.Name + "#" + f.FilePath
			entry := NewFileEntry(name,
				WithSource(SourceKindFile),
				WithContentBytes(f.Data),
				WithLogger(logger),
				WithIsFile(true),
				WithIsDir(false),
			)
			entries = append(entries, entry)
		}
		return entries, nil
	}

	return nil, fmt.Errorf("file %s is not a container (directory or archive)", root.Name)
}
