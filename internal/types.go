package internal

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/google/uuid"
	"strings"
	"text/template"
)

type TemplateType interface {
	Source() string
	Path() string
	Execute(data any) (*bytes.Buffer, error)
	Template() *template.Template
	Functions() template.FuncMap
	AddTemplate(contentString string) error
	ID() string
}

type YutcTemplate struct {
	source         string
	path           string
	template       *template.Template
	templateString string
	functions      template.FuncMap
	id             string
	relativePath   string
}

func (t *YutcTemplate) Functions() template.FuncMap {
	return t.functions
}

func (t *YutcTemplate) SetRelativePath(rootPath string) {
	nRootPath := NormalizeFilepath(rootPath)
	t.relativePath, _ = strings.CutPrefix(t.Path(), nRootPath)
}

func (t *YutcTemplate) RelativePath() string {
	return t.relativePath
}

func NewTemplate(source, path string, funcMap template.FuncMap, auth ...string) (*YutcTemplate, error) {
	var basicAuth, bearerToken string
	var err error
	_id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	id := _id.String()

	switch len(auth) {
	case 0:
		basicAuth = ""
		bearerToken = ""
	case 1:
		basicAuth = auth[0]
		bearerToken = ""
	case 2:
		basicAuth = auth[0]
		bearerToken = auth[1]
	default:
		return nil, fmt.Errorf("invalid number of auth arguments, expected 0, 1 or 2")
	}
	content, err := GetDataFromPath(source, path, basicAuth, bearerToken)
	if err != nil {
		return nil, err
	}
	T, err := template.New(id).Funcs(funcMap).Funcs(sprig.FuncMap()).Parse(content.String())
	if err != nil {
		return nil, err
	}
	return &YutcTemplate{
		id:             id,
		source:         source,
		path:           NormalizeFilepath(path),
		template:       T,
		templateString: content.String(),
		functions:      funcMap,
	}, nil
}

func (t *YutcTemplate) AddTemplate(contentString string) (err error) {
	// Parse the content string as a new template with the same name and functions.
	tempTmpl, err := template.New(t.id).Funcs(t.functions).Parse(contentString)
	var tempOutput bytes.Buffer
	if err != nil {
		return err
	}

	// Execute the new template to ensure it only contains definitions.
	err = tempTmpl.Execute(&tempOutput, nil)
	if err != nil {
		return err
	}

	// Check if the executed template output is not empty, indicating executable code.
	if strings.TrimSpace(tempOutput.String()) != "" {
		return fmt.Errorf("additional templates must definitions only")
	}

	// Parse the content string into the existing template.
	t.template, err = t.template.Parse(contentString)
	return err
}

func (t *YutcTemplate) Source() string {
	return t.source
}

func (t *YutcTemplate) ID() string {
	return t.id
}

func (t *YutcTemplate) Path() string {
	return t.path
}

func (t *YutcTemplate) Execute(data any) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	err := t.template.Execute(buf, data)
	return buf, err
}

func (t *YutcTemplate) Template() *template.Template {
	var err error
	// Make sure than any added templates have not overwritten the original template.
	t.template, err = t.template.Parse(t.templateString)
	if err != nil {
		panic(fmt.Errorf("this should have been caught during NewTemplate call: %w", err))
	}
	return t.template
}
