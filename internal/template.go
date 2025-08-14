package internal

import (
	"bytes"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	yutc "github.com/adam-huganir/yutc/pkg"
)

func BuildTemplate(text string, sharedTemplateBuffers []*bytes.Buffer, name string) (*template.Template, error) {
	var err error
	tmpl := template.New(name).Funcs(
		sprig.FuncMap(),
	).Funcs(template.FuncMap{
		"toYaml":       yutc.ToYaml,
		"fromYaml":     yutc.FromYaml,
		"mustToYaml":   yutc.MustToYaml,
		"mustFromYaml": yutc.MustFromYaml,
		"toToml":       yutc.ToToml,
		"fromToml":     yutc.FromToml,
		"mustToToml":   yutc.MustToToml,
		"mustFromToml": yutc.MustFromToml,
		// "stringMap":    yutc.stringMap,
		"wrapText":     yutc.WrapText,
		"wrapComment":  yutc.WrapComment,
		"fileGlob":     yutc.PathGlob,
		"fileStat":     yutc.PathStat,
		"fileRead":     yutc.FileRead,
		"fileReadN":    yutc.FileReadN,
		"type":         yutc.TypeOf,
		"pathAbsolute": yutc.PathAbsolute,
		"pathIsDir":    yutc.PathIsDir,
		"pathIsFile":   yutc.PathIsFile,
		"pathExists":   yutc.PathExists,
	})
	tmpl = tmpl.Funcs(template.FuncMap{
		"include": yutc.IncludeFun(tmpl, map[string]int{
			"include": 5,
		}),
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

		isDir, err := IsDir(templateFile)
		if err == nil && isDir {
			templates = append(templates, nil) // add a nil entry to make sure our indexes match up
			continue
		}
		source, err := ParseFileStringFlag(templateFile)
		contentBuffer, err := GetDataFromPath(source, templateFile, nil)
		YutcLog.Debug().Msg("Loading from " + source + " template file " + templateFile)
		if err != nil {
			return nil, err
		}
		tmpl, err := BuildTemplate(contentBuffer.String(), sharedTemplateBuffers, templateFile)
		if err != nil {
			return nil, err
		}
		templates = append(templates, tmpl)
	}
	return templates, nil
}
