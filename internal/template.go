package internal

import (
	"bytes"
	"github.com/Masterminds/sprig/v3"
	yutc "github.com/adam-huganir/yutc/pkg"
	"strconv"
	"text/template"
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

func LoadTemplates(settings CLIOptions, sharedTemplateBuffers []*bytes.Buffer) ([]*template.Template, error) {
	var templates []*template.Template
	logger.Debug("Loading " + strconv.Itoa(len(settings.TemplateFiles)) + " template files")
	for _, s := range settings.TemplateFiles {
		logger.Debug("Template file: " + s)
		contentBuffer, err := GetDataFromPath(s)
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
