package data

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"github.com/adam-huganir/yutc/pkg/files"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"

	"dario.cat/mergo"
	"github.com/goccy/go-yaml"
)

// MergeData merges data from a list of data files and returns a map of the merged data.
// The data is merged in the order of the data files, with later files overriding earlier ones.
// Supports files supported by ParseFileStringFlag.
func MergeData(dataFiles []*types.DataFileArg, logger *zerolog.Logger) (map[string]any, error) {
	var err error
	data := make(map[string]any)
	err = mergePaths(dataFiles, &data, logger)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func mergePaths(dataFiles []*types.DataFileArg, data *map[string]any, logger *zerolog.Logger) error {
	for _, dataArg := range dataFiles {

		isDir, _ := afero.IsDir(files.Fs, dataArg.Path)
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
		err = yaml.Unmarshal(contentBuffer.Bytes(), &dataPartial)
		if err != nil {
			return err
		}

		// If a top-level key is specified, nest the data under that key
		if dataArg.Key != "" {
			logger.Debug().Msg(fmt.Sprintf("Nesting data under top-level key: %s", dataArg.Key))
			dataPartial = map[string]any{dataArg.Key: dataPartial}
		}

		err = mergo.Merge(data, dataPartial, mergo.WithOverride)
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadSharedTemplates reads from a list of shared template files and returns a list of buffers with the contents
func LoadSharedTemplates(templates []string, logger *zerolog.Logger) []*bytes.Buffer {
	var sharedTemplateBuffers []*bytes.Buffer
	for _, template := range templates {
		isDir, _ := afero.IsDir(files.Fs, template)
		if isDir {
			continue
		}
		source, _ := files.ParseFileStringFlag(template)
		logger.Debug().Msg("Loading from " + source + " shared template file " + template)
		contentBuffer, err := files.GetDataFromPath(source, template, "", "")
		if err != nil {
			panic(err)
		}
		sharedTemplateBuffers = append(sharedTemplateBuffers, contentBuffer)
	}
	return sharedTemplateBuffers
}

func LoadTemplates(
	ctx context.Context,
	templatePaths []string,
	tempDir string,
	logger *zerolog.Logger,
) (
	[]string,
	error,
) {
	templateFiles, _ := files.ResolvePaths(ctx, templatePaths, tempDir, logger)
	// this sort will help us later when we make assumptions about if folders already exist
	slices.SortFunc(templateFiles, func(a, b string) int {
		aIsShorter := len(a) < len(b)
		if aIsShorter {
			return -1
		}
		return 1
	})

	logger.Debug().Msg(fmt.Sprintf("Found %d template files", len(templateFiles)))
	for _, templateFile := range templateFiles {
		logger.Trace().Msg("  - " + templateFile)
	}
	return templateFiles, nil
}

func LoadDataFiles(dataFiles []*types.DataFileArg, tempDir string, logger *zerolog.Logger) ([]*types.DataFileArg, error) {
	dataPathsOnly := make([]string, len(dataFiles))
	for idx, dataFile := range dataFiles {
		dataPathsOnly[idx] = dataFile.Path
	}
	paths, err := files.ResolvePaths(context.Background(), dataPathsOnly, tempDir, logger)
	if err != nil {
		return nil, err
	}
	for idx, newPath := range paths {
		dataFiles[idx].Path = newPath
	}

	return dataFiles, nil
}

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

func ParseTemplatePaths(rd *types.RunData, templatePaths []string) error {
	rd.TemplatePaths = append(rd.TemplatePaths, templatePaths...)
	return nil

}

func ParseCommonTemplateFiles(rd *types.RunData, commonTemplateFiles []string) error {
	rd.CommonTemplateFiles = append(rd.CommonTemplateFiles, commonTemplateFiles...)
	return nil
}
