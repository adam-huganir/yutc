package internal

import (
	"github.com/Masterminds/sprig/v3"
	"text/template"
)

func BuildTemplate(text string) (*template.Template, error) {
	tmpl, err := template.New("template").Funcs(
		sprig.FuncMap()).Parse(text)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}
