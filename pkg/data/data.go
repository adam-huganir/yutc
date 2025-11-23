package data

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"slices"
	"strings"

	"github.com/adam-huganir/yutc/pkg/config"
	"github.com/adam-huganir/yutc/pkg/files"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"

	"dario.cat/mergo"
	"github.com/goccy/go-yaml"
)

// ParseDataFileArg parses a data file argument which can be in two formats:
// 1. Simple path: "./my_file.yaml"
// 2. With key: "key=Secrets,src=./my_secrets.yaml"
func ParseDataFileArg(arg string) (*types.DataFileArg, error) {
	// Check if the argument contains the structured format
	hasKey := strings.Contains(arg, "key=")
	hasSrc := strings.Contains(arg, "src=")

	// If either key= or src= is present, we expect the structured format. if an equals is in there otherwise we just
	// take that as the filename
	if hasKey || hasSrc {
		// Use CSV reader to properly parse comma-separated key=value pairs
		dataArg := &types.DataFileArg{}
		data, err := mapFromKeyValueOption(arg)
		if err != nil {
			return nil, err
		}

		for key, value := range data {
			switch key {
			case "key":
				dataArg.Key = value
			case "src":
				dataArg.Path = value
			default:
				return nil, fmt.Errorf("invalid data argument format with unknown parameter %s: %s", key, arg)
			}
		}
		if dataArg.Path == "" {
			return nil, fmt.Errorf("missing 'src' parameter in data argument: %s", arg)
		}

		return dataArg, nil
	}

	// Simple format - just a path
	return &types.DataFileArg{
		Key:  "",
		Path: arg,
	}, nil
}

func mapFromKeyValueOption(arg string) (map[string]string, error) {
	reader := csv.NewReader(strings.NewReader(arg))
	reader.TrimLeadingSpace = true

	records, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to parse data argument: %w", err)
	}

	data := make(map[string]string)

	for _, part := range records {
		if !strings.Contains(part, "=") {
			return nil, fmt.Errorf("invalid data argument format, no argument provided in %s: %s", part, arg)
		}
		part = strings.TrimSpace(part)
		prefix := part[:strings.Index(part, "=")]
		value := part[strings.Index(part, "=")+1:]
		data[prefix] = value
	}
	return data, nil
}

// MergeData merges data from a list of data files and returns a map of the merged data.
// The data is merged in the order of the data files, with later files overriding earlier ones.
// Supports files supported by ParseFileStringFlag.
func MergeData(ctx context.Context) (map[string]any, error) {
	var err error
	logger := config.GetLogger(ctx)
	dataFiles := config.GetRunData(ctx).DataFiles
	data := make(map[string]any)
	err = mergePaths(dataFiles, &data, logger)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func mergePaths(dataFiles []*types.DataFileArg, data *map[string]any, logger zerolog.Logger) error {
	for _, dataArg := range dataFiles {

		isDir, err := afero.IsDir(files.Fs, dataArg.Path)
		if isDir {
			continue
		}
		source, err := files.ParseFileStringFlag(dataArg.Path)
		if err != nil {
			return err
		}
		logger.Debug().Msg("Loading from " + source + " data file " + dataArg.Path)
		contentBuffer, err := files.GetDataFromPath(source, dataArg.Path, nil)
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
func LoadSharedTemplates(templates []string, logger zerolog.Logger) []*bytes.Buffer {
	var sharedTemplateBuffers []*bytes.Buffer
	for _, template := range templates {
		isDir, err := afero.IsDir(files.Fs, template)
		if isDir {
			continue
		}
		source, err := files.ParseFileStringFlag(template)
		logger.Debug().Msg("Loading from " + source + " shared template file " + template)
		contentBuffer, err := files.GetDataFromPath(source, template, nil)
		if err != nil {
			panic(err)
		}
		sharedTemplateBuffers = append(sharedTemplateBuffers, contentBuffer)
	}
	return sharedTemplateBuffers
}

func LoadTemplates(
	ctx context.Context,
) (
	[]string,
	error,
) {
	settings := config.GetSettings(ctx)
	logger := config.GetLogger(ctx)

	templateFiles, _ := files.ResolvePaths(ctx, settings.TemplatePaths)
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

func LoadDataFiles(ctx context.Context) ([]*types.DataFileArg, error) {
	tempDir := config.GetTempDir(ctx)
	logger := config.GetLogger(ctx)
	dataFiles := config.GetRunData(ctx).DataFiles

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

func ParseDataFiles(rd *types.RunData, dataFiles []string) error {
	for _, dataFileArg := range dataFiles {
		dataArg, err := ParseDataFileArg(dataFileArg)
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
