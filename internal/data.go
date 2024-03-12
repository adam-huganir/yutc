package internal

import (
	"bytes"
	"strconv"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"
)

func MergeData(settings CLIOptions, buffers ...*bytes.Buffer) (map[any]any, error) {
	var err error

	data := make(map[any]any)
	logger.Trace("Loading " + strconv.Itoa(len(settings.DataFiles)) + " data files")

	if settings.StdinFirst {
		if buffers != nil {
			err = mergeStdIn(buffers, &data)
			if err != nil {
				return nil, err
			}
		}
		err = mergePaths(settings, &data)
		if err != nil {
			return nil, err
		}
	} else {
		err := mergePaths(settings, &data)
		if err != nil {
			return nil, err
		}
		if buffers != nil {
			err = mergeStdIn(buffers, &data)
			if err != nil {
				return nil, err
			}
		}

	}
	return data, nil
}

func mergePaths(settings CLIOptions, data *map[any]any) error {
	for _, s := range settings.DataFiles {
		logger.Debug("Data file: " + s)
		path, err := ParseFileStringFlag(s)
		if err != nil {
			return err
		}
		logger.Debug("Data file path: " + path.String())
		contentBuffer, err := GetDataFromPath(path.String())
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

func mergeStdIn(buffers []*bytes.Buffer, data *map[any]any) error {
	for _, b := range buffers {
		if b == nil {
			continue
		}
		err := yaml.Unmarshal(b.Bytes(), *data)
		if err != nil {
			return err
		}
		dataPartial := make(map[any]any)
		err = mergo.Merge(data, dataPartial, mergo.WithOverride)
		if err != nil {
			return err
		}
	}
	return nil
}
