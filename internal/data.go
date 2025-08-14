package internal

import (
	"bytes"
	"errors"
	"github.com/spf13/afero"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"
)

// MergeData merges data from a list of data files and returns a map of the merged data.
// The data is merged in the order of the data files, with later files overriding earlier ones.
// Supports files supported by ParseFileStringFlag.
func MergeData(dataFiles []string) (map[string]any, error) {
	var err error
	var data map[string]any
	err = mergePaths(dataFiles, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func mergePaths(dataFiles []string, data *map[string]any) error {
	for _, arg := range dataFiles {
		isDir, err := afero.IsDir(Fs, arg)
		if isDir {
			continue
		}
		source, err := ParseFileStringFlag(arg)
		if err != nil {
			return err
		}
		YutcLog.Debug().Msg("Loading from " + source + " data file " + arg)
		contentBuffer, err := GetDataFromPath(source, arg, nil)
		if err != nil {
			return err
		}
		dataPartial := make(map[string]any)
		err = yaml.Unmarshal(contentBuffer.Bytes(), &dataPartial)
		if err != nil {
			return err
		}
		err = mergo.Merge(data, dataPartial, mergo.WithOverride)
		if err != nil {
			return err
		}
	}
	return nil
}

// ParseFileStringFlag determines the source of a file string flag based on format and returns the source
// as a string, or an error if the source is not supported. Currently, supports "file", "url", and "stdin" (as `-`).
func ParseFileStringFlag(v string) (string, error) {
	if !strings.Contains(v, "://") {
		if v == "-" {
			return "stdin", nil
		}
		_, err := filepath.Abs(v)
		if err != nil {
			return "", err
		}
		return "file", nil
	}
	if v == "-" {
		return "stdin", nil
	}
	allowedUrlPrefixes := []string{"http://", "https://"}
	for _, prefix := range allowedUrlPrefixes {
		if strings.HasPrefix(v, prefix) {
			return "url", nil
		}
	}
	return "", errors.New("unsupported scheme/source for input: " + v)
}

// LoadSharedTemplates reads from a list of shared template files and returns a list of buffers with the contents
func LoadSharedTemplates(templates []string) []*bytes.Buffer {
	var sharedTemplateBuffers []*bytes.Buffer
	for _, template := range templates {
		isDir, err := afero.IsDir(Fs, template)
		if isDir {
			continue
		}
		source, err := ParseFileStringFlag(template)
		YutcLog.Debug().Msg("Loading from " + source + " shared template file " + template)
		contentBuffer, err := GetDataFromPath(source, template, nil)
		if err != nil {
			panic(err)
		}
		sharedTemplateBuffers = append(sharedTemplateBuffers, contentBuffer)
	}
	return sharedTemplateBuffers
}
