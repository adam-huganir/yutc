package data

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/adam-huganir/yutc/internal/files"
	"github.com/adam-huganir/yutc/internal/logging"
	"github.com/adam-huganir/yutc/internal/types"
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
			if key == "key" {
				dataArg.Key = value
			} else if key == "src" {
				dataArg.Path = value
			} else {
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

// CountDataRecursables counts the number of recursable (directory or archive) data files
func CountDataRecursables(dataFiles []string) (int, error) {
	recursables := 0
	for _, dataFileArg := range dataFiles {
		dataArg, err := ParseDataFileArg(dataFileArg)
		if err != nil {
			return recursables, err
		}

		source, err := files.ParseFileStringFlag(dataArg.Path)
		if source != "file" {
			if source == "url" {
				if files.IsArchive(dataArg.Path) {
					recursables++
				}
			}
			continue
		}
		isDir, err := files.IsDir(dataArg.Path)
		if err != nil {
			return recursables, err
		} else if isDir || files.IsArchive(dataArg.Path) {
			recursables++
		}
	}
	return recursables, nil
}

// MergeData merges data from a list of data files and returns a map of the merged data.
// The data is merged in the order of the data files, with later files overriding earlier ones.
// Supports files supported by ParseFileStringFlag.
func MergeData(dataFiles []*types.DataFileArg) (map[string]any, error) {
	var err error
	data := make(map[string]any)
	err = mergePaths(dataFiles, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func mergePaths(dataFiles []*types.DataFileArg, data *map[string]any) error {
	for _, dataArg := range dataFiles {

		isDir, err := afero.IsDir(files.Fs, dataArg.Path)
		if isDir {
			continue
		}
		source, err := files.ParseFileStringFlag(dataArg.Path)
		if err != nil {
			return err
		}
		logging.YutcLog.Debug().Msg("Loading from " + source + " data file " + dataArg.Path)
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
			logging.YutcLog.Debug().Msg(fmt.Sprintf("Nesting data under top-level key: %s", dataArg.Key))
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
func LoadSharedTemplates(templates []string) []*bytes.Buffer {
	var sharedTemplateBuffers []*bytes.Buffer
	for _, template := range templates {
		isDir, err := afero.IsDir(files.Fs, template)
		if isDir {
			continue
		}
		source, err := files.ParseFileStringFlag(template)
		logging.YutcLog.Debug().Msg("Loading from " + source + " shared template file " + template)
		contentBuffer, err := files.GetDataFromPath(source, template, nil)
		if err != nil {
			panic(err)
		}
		sharedTemplateBuffers = append(sharedTemplateBuffers, contentBuffer)
	}
	return sharedTemplateBuffers
}
