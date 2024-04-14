package internal

import (
	"bytes"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	yutc "github.com/adam-huganir/yutc/pkg"
)

func BuildTemplate(text string, sharedTemplateBuffers []*bytes.Buffer) (*template.Template, error) {
	var err error
	tmpl := template.New("template").Funcs(
		sprig.FuncMap(),
	).Funcs(template.FuncMap{
		"toYaml":       yutc.ToYaml,
		"fromYaml":     yutc.FromYaml,
		"mustToYaml":   yutc.MustToYaml,
		"mustFromYaml": yutc.MustFromYaml,
		// "stringMap":    yutc.stringMap,
		"wrapComment": yutc.WrapComment,
	})
	for _, sharedTemplateBuffer := range sharedTemplateBuffers {
		tmpl, err = tmpl.Parse(sharedTemplateBuffer.String())
		if err != nil {
			return nil, err
		}

	}
	tmpl, err = tmpl.Parse(text)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

func LoadTemplates(templateFiles []string, sharedTemplateBuffers []*bytes.Buffer) ([]*template.Template, error) {
	var templates []*template.Template
	YutcLog.Debug().Msg("Loading " + strconv.Itoa(len(templateFiles)) + " template files")
	for _, templateFile := range templateFiles {
		source, err := ParseFileStringFlag(templateFile)
		contentBuffer, err := GetDataFromPath(source, templateFile)
		YutcLog.Debug().Msg("Loading from " + source + " template file " + templateFile)
		if err != nil {
			return nil, err
		}
		tmpl, err := BuildTemplate(contentBuffer.String(), sharedTemplateBuffers)
		if err != nil {
			return nil, err
		}
		templates = append(templates, tmpl)
	}
	return templates, nil
}
