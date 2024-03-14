package internal

import (
	"bytes"
	"errors"
	"github.com/adam-huganir/yutc/pkg/LoggingUtils"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"
)

var logger = LoggingUtils.GetLogHandler()

func MergeData(dataFiles []string) (map[any]any, error) {
	var err error

	data := make(map[any]any)
	logger.Trace("Loading " + strconv.Itoa(len(dataFiles)) + " data files")

	err = mergePaths(dataFiles, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func mergePaths(dataFiles []string, data *map[any]any) error {
	for _, arg := range dataFiles {
		source, err := ParseFileStringFlag(arg)
		if err != nil {
			return err
		}
		logger.Debug("Loading from " + source + " data file " + arg)
		contentBuffer, err := GetDataFromPath(source, arg)
		if err != nil {
			return err
		}
		dataPartial := make(map[any]any)
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
		logger.Debug("Loading from " + source + " shared template file " + template)
		contentBuffer, err := GetDataFromPath(source, template)
		if err != nil {
			panic(err)
		}
		sharedTemplateBuffers = append(sharedTemplateBuffers, contentBuffer)
	}
	return sharedTemplateBuffers
}
