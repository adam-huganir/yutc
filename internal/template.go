package internal

import (
	"strconv"
)

func LoadTemplates(templateFiles []string, basicAuth, bearerToken string) ([]*YutcTemplate, error) {
	templateContents, err := LoadFiles(templateFiles, basicAuth, bearerToken)
	if err != nil {
		return nil, err
	}

	var templates []*YutcTemplate
	YutcLog.Debug().Msg("Loading " + strconv.Itoa(len(templateFiles)) + " template files")
	for _, templateContent := range templateContents {
		YutcLog.Debug().Msg("Loading from " + templateContent.Source + " template file " + templateContent.Path)
		tmpl, err := NewTemplate(templateContent.Source, templateContent.Path, funcMap, basicAuth, bearerToken)
		if err != nil {
			return nil, err
		}
		templates = append(templates, tmpl)
	}
	return templates, nil
}

func LoadFiles(files []string, basicAuth, bearerToken string) ([]*FileData, error) {

	var fileData []*FileData
	var loopFiles []string
	for _, file := range files {
		// first, recurse through everything if needed
		isRecursive, err := isRecursable(file)
		if err != nil {
			return nil, err
		}
		if isRecursive {
			loopFiles = WalkDir(file)
		} else {
			loopFiles = []string{file}
		}

		for _, f := range loopFiles {
			isDir, err := IsDir(f)
			if err == nil && isDir {
				continue
			}
			source, err := ParseFileStringFlag(f)
			if err != nil {
				return nil, err
			}
			content, err := GetDataFromPath(source, f, basicAuth, bearerToken)
			fileData = append(fileData, &FileData{
				Path:       f,
				Source:     source,
				ReadWriter: content,
			})
		}
	}
	return fileData, nil
}

func isRecursable(path string) (bool, error) {
	inputIsRecursive, err := IsDir(path)
	if !inputIsRecursive {
		inputIsRecursive = IsArchive(path)
	}
	return inputIsRecursive, err
}
