// Package data handles data loading, merging, and manipulation for yutc templates.
package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/adam-huganir/yutc/pkg/files"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"

	"dario.cat/mergo"
	"github.com/goccy/go-yaml"
)

// MergeData merges data from a list of data files and returns a map of the merged data.
// The data is merged in the order of the data files, with later files overriding earlier ones.
// Supports files supported by ParseFileStringFlag.
func MergeData(dataFiles []*types.DataFileArg, helmMode bool, logger *zerolog.Logger) (map[string]any, error) {
	var err error
	data := make(map[string]any)
	err = mergePaths(dataFiles, data, helmMode, logger)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func mergePaths(dataFiles []*types.DataFileArg, data map[string]any, helmMode bool, logger *zerolog.Logger) error {
	// since some of helms data structures are go structs, when the chart file is accessed through templates
	// it uses the struct casing rather than the yaml casing. this adjusts for that. for right now we only do this
	// for Chart
	specialHelmKeys := []string{"Chart"}
	for _, dataArg := range dataFiles {

		isDir, err := afero.IsDir(files.Fs, dataArg.Path)
		if err != nil {
			return err
		}
		if isDir {
			continue
		}
		source, err := files.ParseFileStringFlag(dataArg.Path)
		if err != nil {
			return err
		}
		logger.Debug().Msg("Loading from " + source + " data file " + dataArg.Path)
		contentBuffer, err := files.GetDataFromPath(source, dataArg.Path, "", "")
		if err != nil {
			return err
		}
		dataPartial := make(map[string]any)

		switch strings.ToLower(path.Ext(dataArg.Path)) {
		case ".toml":
			err = toml.Unmarshal(contentBuffer.Bytes(), &dataPartial)
		// originally i had used yaml to parse the json, but then thought that the expected behavior for giving invalid
		// json would be to fail, even if it was valid yaml
		case ".json":
			err = json.Unmarshal(contentBuffer.Bytes(), &dataPartial)
		default:
			err = yaml.Unmarshal(contentBuffer.Bytes(), &dataPartial)
		}
		if err != nil {
			return errors.Wrapf(err, "unable to load data file %s", dataArg.Path)
		}

		// If a top-level key is specified, nest the data under that key
		if dataArg.Key != "" {
			logger.Debug().Msg(fmt.Sprintf("Nesting data for %s under top-level key: %s", dataArg.Path, dataArg.Key))
			if helmMode && slices.Contains(specialHelmKeys, dataArg.Key) {
				logger.Debug().Msg(fmt.Sprintf("Applying helm key transformation for %s", dataArg.Key))
				dataPartial = KeysToPascalCase(dataPartial)
			}
			dataPartial = map[string]any{dataArg.Key: dataPartial}
		}

		err = mergo.Merge(&data, dataPartial, mergo.WithOverride)
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadSharedTemplates reads from a list of shared template files and returns a list of buffers with the contents
func LoadSharedTemplates(templates []string, logger *zerolog.Logger) ([]*bytes.Buffer, error) {
	var sharedTemplateBuffers []*bytes.Buffer
	for _, template := range templates {
		isDir, err := afero.IsDir(files.Fs, template)
		if err != nil {
			return nil, err
		}
		if isDir {
			continue
		}
		source, err := files.ParseFileStringFlag(template)
		if err != nil {
			return nil, err
		}
		logger.Debug().Msg("Loading from " + source + " shared template file " + template)
		contentBuffer, err := files.GetDataFromPath(source, template, "", "")
		if err != nil {
			return nil, err
		}
		sharedTemplateBuffers = append(sharedTemplateBuffers, contentBuffer)
	}
	return sharedTemplateBuffers, nil
}

// LoadTemplates resolves template paths and returns a sorted list of template file paths.
// It resolves directories, archives, and URLs to actual file paths and sorts them.
func LoadTemplates(
	templatePaths []string,
	tempDir string,
	logger *zerolog.Logger,
) (
	[]string,
	error,
) {
	templateFiles, err := files.ResolvePaths(templatePaths, tempDir, logger)
	if err != nil {
		return nil, err
	}
	// this sort will help us later when we make assumptions about if folders already exist
	slices.Sort(templateFiles)

	logger.Debug().Msg(fmt.Sprintf("Found %d template files", len(templateFiles)))
	for _, templateFile := range templateFiles {
		logger.Trace().Msg("  - " + templateFile)
	}
	return templateFiles, nil
}

// LoadDataFiles resolves data file paths (directories, archives, URLs) to actual file paths.
// Returns an updated list of DataFileArg with resolved paths.
func LoadDataFiles(dataFiles []*types.DataFileArg, tempDir string, logger *zerolog.Logger) ([]*types.DataFileArg, error) {
	dataPathsOnly := make([]string, len(dataFiles))
	for idx, dataFile := range dataFiles {
		dataPathsOnly[idx] = dataFile.Path
	}
	paths, err := files.ResolvePaths(dataPathsOnly, tempDir, logger)
	if err != nil {
		return nil, err
	}
	for idx, newPath := range paths {
		dataFiles[idx].Path = newPath
	}

	return dataFiles, nil
}

// ParseDataFiles parses raw data file arguments and populates the RunData structure.
func ParseDataFiles(rd *types.RunData, dataFiles []string) error {
	for _, dataFileArg := range dataFiles {
		dataArg, err := files.ParseDataFileArg(dataFileArg)
		if err != nil {
			return err
		}
		rd.DataFiles = append(rd.DataFiles, dataArg)
	}
	return nil
}

// ParseTemplatePaths adds template paths to the RunData structure.
func ParseTemplatePaths(rd *types.RunData, templatePaths []string) error {
	rd.TemplatePaths = append(rd.TemplatePaths, templatePaths...)
	return nil

}

// ParseCommonTemplateFiles adds common template file paths to the RunData structure.
func ParseCommonTemplateFiles(rd *types.RunData, commonTemplateFiles []string) error {
	rd.CommonTemplateFiles = append(rd.CommonTemplateFiles, commonTemplateFiles...)
	return nil
}
