package yutc

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

func BuildTemplate(text string, sharedTemplateBuffers []*bytes.Buffer, name string, strict bool) (*template.Template, error) {
	var err error
	var onError string
	if strict {
		onError = "error"
	} else {
		onError = "zero"
	}
	tmpl := template.New(name).Option("missingkey=" + onError).Funcs(
		sprig.FuncMap(),
	).Funcs(template.FuncMap{
		"toYaml":       ToYaml,
		"fromYaml":     FromYaml,
		"mustToYaml":   MustToYaml,
		"yamlOptions":  SetYamlEncodeOptions,
		"mustFromYaml": MustFromYaml,
		"toToml":       ToToml,
		"fromToml":     FromToml,
		"mustToToml":   MustToToml,
		"mustFromToml": MustFromToml,
		// "stringMap": c.stringMap,
		"wrapText":     WrapText,
		"wrapComment":  WrapComment,
		"fileGlob":     PathGlob,
		"fileStat":     PathStat,
		"fileRead":     FileRead,
		"fileReadN":    FileReadN,
		"type":         TypeOf,
		"pathAbsolute": PathAbsolute,
		"pathIsDir":    PathIsDir,
		"pathIsFile":   PathIsFile,
		"pathExists":   PathExists,
	})
	includedNames := make(map[string]int)
	tmpl = tmpl.Funcs(template.FuncMap{
		"include": IncludeFun(tmpl, includedNames),
		"tpl":     TplFun(tmpl, includedNames, strict),
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
