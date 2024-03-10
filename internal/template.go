package internal

import (
	"github.com/Masterminds/sprig/v3"
	yutc "github.com/adam-huganir/yutc/pkg"
	"strconv"
	"text/template"
)

func BuildTemplate(text string) (*template.Template, error) {
	tmpl, err := template.New("template").Funcs(
		sprig.FuncMap(),
	).Funcs(template.FuncMap{
		"toYaml":       yutc.ToYaml,
		"fromYaml":     yutc.FromYaml,
		"mustToYaml":   yutc.MustToYaml,
		"mustFromYaml": yutc.MustFromYaml,
	}).Parse(text)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

func LoadTemplates(settings CLIOptions) ([]*template.Template, error) {
	var templates []*template.Template
	logger.Debug("Loading " + strconv.Itoa(len(settings.TemplateFiles)) + " template files")
	for _, s := range settings.TemplateFiles {
		logger.Debug("Template file: " + s)
		contentBuffer, err := GetDataFromPath(s)
		if err != nil {
			return nil, err
		}
		tmpl, err := BuildTemplate(contentBuffer.String())
		if err != nil {
			return nil, err
		}
		templates = append(templates, tmpl)
	}
	return templates, nil
}
