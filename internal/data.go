package internal

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"
)

func MergeData(dataFiles []string) (map[string]any, error) {
	var err error

	data := make(map[string]any)
	err = mergePaths(dataFiles, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func mergePaths(dataFiles []string, data *map[string]any) error {
	for _, arg := range dataFiles {
		source, err := ParseFileStringFlag(arg)
		if err != nil {
			return err
		}
		YutcLog.Debug().Msg("Loading from " + source + " data file " + arg)
		contentBuffer, err := GetDataFromPath(source, arg)
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
	allowedPrefixes := []string{"http://", "https://"}
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(v, prefix) {
			return "url", nil
		}
	}
	return "", errors.New("unsupported scheme/source for input: " + v)
}

func LoadSharedTemplates(templates []string) []*bytes.Buffer {
	var sharedTemplateBuffers []*bytes.Buffer
	for _, template := range templates {
		source, err := ParseFileStringFlag(template)
		YutcLog.Debug().Msg("Loading from " + source + " shared template file " + template)
		contentBuffer, err := GetDataFromPath(source, template)
		if err != nil {
			panic(err)
		}
		sharedTemplateBuffers = append(sharedTemplateBuffers, contentBuffer)
	}
	return sharedTemplateBuffers
}
