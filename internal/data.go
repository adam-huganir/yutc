package internal

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/spf13/afero"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"
)

// GatherData merges data from a list of data files and returns a map of the merged data.
// The data is merged in the order of the data files, with later files overriding earlier ones.
// Supports files supported by ParseFileStringFlag.
func GatherData(dataFiles []string, appendMode bool, basicAuth, bearerToken string) (any, reflect.Kind, error) {
	var finalData any
	var tempData any
	var lastType reflect.Kind
	for _, dataFile := range dataFiles {
		isDir, err := afero.IsDir(Fs, dataFile)
		if isDir {
			continue
		}
		source, err := ParseFileStringFlag(dataFile)
		if err != nil {
			return nil, lastType, err
		}
		YutcLog.Debug().Msgf("Loading from %s data file %s", source, dataFile)
		contentBuffer, err := GetDataFromPath(source, dataFile, basicAuth, bearerToken)
		if err != nil {
			return nil, lastType, err
		}
		if finalData == nil {
			err = yaml.Unmarshal(contentBuffer.Bytes(), &finalData)
			if err != nil {
				return nil, reflect.Invalid, err
			}
			lastType = reflect.TypeOf(finalData).Kind()
		} else {
			tempData, lastType, err = updateData(contentBuffer, finalData, lastType, appendMode)
			if err != nil {
				return nil, lastType, err
			}
			finalData = &tempData
		}
	}
	return finalData, lastType, nil
}

func initData(d any) any {

	dataType := reflect.TypeOf(d).Kind()
	switch dataType {
	case reflect.Map:
		// coerce to a string map for serialization compatability
		anyMap, err := coerceToStringMap(d)
		if err != nil {
			return err
		}
		return anyMap
	case reflect.Slice:
		anyArray, ok := d.([]any)
		if !ok {
			return fmt.Errorf("data file is not a list, this shouldnt be possible")
		}
		return anyArray
	default:
		{
			// at this junction probably a scalar of some sort
			return d
		}
	}
}

func updateData(contentBuffer *bytes.Buffer, currentData any, lastType reflect.Kind, appendMode bool) (any, reflect.Kind, error) {
	var dataAny interface{}
	var anyMap, mergedMap map[string]interface{}
	var anyArray, mergedArray []interface{}
	var ok bool
	err := yaml.Unmarshal(contentBuffer.Bytes(), &dataAny)
	if err != nil {
		return nil, reflect.Invalid, err
	}
	typeOfData := reflect.TypeOf(dataAny).Kind()

	if typeOfData == lastType {
		if typeOfData == reflect.Map {
			anyMap, err = coerceToStringMap(dataAny)
			if err != nil {
				return nil, reflect.Invalid, err
			}
			mergedMap = currentData.(map[string]interface{})
			err = mergo.Merge(&mergedMap, anyMap, mergo.WithOverride)
			if err != nil {
				return nil, reflect.Invalid, err
			}
			return mergedMap, typeOfData, nil
		} else if typeOfData == reflect.Slice {
			if !appendMode {
				return nil, reflect.Invalid, fmt.Errorf("cannot merge lists without append mode")
			}
			anyArray, ok = dataAny.([]interface{})
			if !ok {
				return nil, reflect.Invalid, fmt.Errorf("data file is not a list, this shouldnt be possible")
			}
			mergedArray, ok = currentData.([]interface{})
			if !ok {
				return nil, reflect.Invalid, fmt.Errorf("current data is not a list, this shouldnt be possible")
			}
			mergedArray = append(mergedArray, anyArray...)
			return mergedArray, typeOfData, nil
		} else {
			YutcLog.Debug().Msgf("Cannot merge data of type %s", typeOfData)
			return nil, reflect.Invalid, fmt.Errorf("cannot merge data of type %s", typeOfData)
		}
	} else {
		YutcLog.Error().Msgf("Cannot merge data of different types %s and %s", lastType, typeOfData)
		return nil, reflect.Invalid, fmt.Errorf("cannot merge data of different types %s and %s", lastType, typeOfData)
	}
}

// coerceToStringMap coerces a map[interface{}]interface{} to a map[string]interface{} so that it can be serialized
// to some output that does not allow it (e.g. json).
func coerceToStringMap(dataAny any) (map[string]any, error) {
	var dataMap map[string]interface{}
	dataMap, ok := dataAny.(map[string]interface{})
	if !ok {
		dataMap = make(map[string]interface{})
		dataAnyMap, ok := dataAny.(map[interface{}]interface{})
		if !ok {
			panic("we shouldn't be here")
		}
		for k, v := range dataAnyMap {
			dataMap[fmt.Sprintf("%v", k)] = v
		}
	}
	return dataMap, nil
}

// ParseFileStringFlag determines the source of a file string flag based on format and returns the source
// as a string, or an error if the source is not supported. Currently, it supports "file", "url", and "stdin" (as `-`).
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
