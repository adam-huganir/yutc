package internal

import "text/template"

func ParseTmpl(template *template.Template, name, text string) (*template.Template, error) {
	return template.New(name).Parse(text)
}
