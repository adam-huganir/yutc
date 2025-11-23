package template

import (
	"bytes"
	"strconv"
	"text/template"

	"github.com/adam-huganir/yutc/pkg/files"
	"github.com/rs/zerolog"
)

// LoadTemplates loads template files and parses them with shared templates and custom functions.
// It returns a slice of templates, with nil entries for directories to maintain index alignment.
func LoadTemplates(templateFiles []string, sharedTemplateBuffers []*bytes.Buffer, strict bool, logger *zerolog.Logger) ([]*template.Template, error) {
	var templates []*template.Template
	logger.Debug().Msg("Loading " + strconv.Itoa(len(templateFiles)) + " template files")
	for _, templateFile := range templateFiles {

		isDir, err := files.IsDir(templateFile)
		if err == nil && isDir {
			templates = append(templates, nil) // add a nil entry to make sure our indexes match up
			continue
		}
		source, err := files.ParseFileStringFlag(templateFile)
		if err != nil {
			return nil, err
		}
		contentBuffer, err := files.GetDataFromPath(source, templateFile, "", "")
		logger.Debug().Msg("Loading from " + source + " template file " + templateFile)
		if err != nil {
			return nil, err
		}
		tmpl, err := BuildTemplate(contentBuffer.String(), sharedTemplateBuffers, templateFile, strict)
		if err != nil {
			return nil, err
		}
		templates = append(templates, tmpl)
	}
	return templates, nil
}
