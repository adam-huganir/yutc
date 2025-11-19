package internal

import (
	"bytes"
	"strconv"
	"text/template"

	"github.com/adam-huganir/yutc/internal/files"
	"github.com/adam-huganir/yutc/internal/logging"
	yutc "github.com/adam-huganir/yutc/pkg"
)

func LoadTemplates(templateFiles []string, sharedTemplateBuffers []*bytes.Buffer, strict bool) ([]*template.Template, error) {
	var templates []*template.Template
	logging.YutcLog.Debug().Msg("Loading " + strconv.Itoa(len(templateFiles)) + " template files")
	for _, templateFile := range templateFiles {

		isDir, err := files.IsDir(templateFile)
		if err == nil && isDir {
			templates = append(templates, nil) // add a nil entry to make sure our indexes match up
			continue
		}
		source, err := files.ParseFileStringFlag(templateFile)
		contentBuffer, err := files.GetDataFromPath(source, templateFile, nil)
		logging.YutcLog.Debug().Msg("Loading from " + source + " template file " + templateFile)
		if err != nil {
			return nil, err
		}
		tmpl, err := yutc.BuildTemplate(contentBuffer.String(), sharedTemplateBuffers, templateFile, strict)
		if err != nil {
			return nil, err
		}
		templates = append(templates, tmpl)
	}
	return templates, nil
}
