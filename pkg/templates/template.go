package templates

import (
	"bytes"
	"fmt"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/adam-huganir/yutc/pkg/data"
	"github.com/adam-huganir/yutc/pkg/quote"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
)

// TemplateSet holds a single template with all parsed templates and their source information.
type TemplateSet struct {
	Template      *template.Template
	TemplateItems []*data.FileArg
}

// LoadTemplateSet loads template data and parses them with shared templates and custom functions.
// Following Helm's approach: creates ONE template object, parses all data into it.
func LoadTemplateSet(templateFiles []*data.FileArg, sharedTemplateBuffers []*data.FileArg, strict bool, logger *zerolog.Logger) (*TemplateSet, error) {
	logger.Debug().Msg("Loading " + strconv.Itoa(len(templateFiles)) + " template data")

	t, err := InitTemplate(sharedTemplateBuffers, strict)
	if err != nil {
		return nil, err
	}

	// Parse all template data into the same template object
	var templateItems []*data.FileArg
	for _, templateFile := range templateFiles {
		if isDir, err := templateFile.IsDir(); err == nil && !isDir {
			templateItems = append(templateItems, templateFile)
		} else if err != nil {
			return nil, err
		}
		children := templateFile.AllChildren()
		if children != nil {
			for _, c := range children {
				if isDir, err := c.IsDir(); err == nil && !isDir {
					templateItems = append(templateItems, c)
				} else if err != nil {
					return nil, err
				}
			}
		}
		logger.Debug().Msgf("Loading from %s template file %s", templateFile.Source, templateFile.Path)
	}

	t, err = ParseTemplateItems(t, templateItems)
	if err != nil {
		return nil, err
	}
	return &TemplateSet{
		Template:      t,
		TemplateItems: templateItems,
	}, nil
}

// ParseTemplateItems parses template data into the same template object.
func ParseTemplateItems(t *template.Template, items []*data.FileArg) (*template.Template, error) {
	var err error
	for _, item := range items {
		if !item.Content.Read {
			err = item.Load()
			if err != nil {
				return nil, fmt.Errorf("unable to load template file %s from %s: %w", item.Path, item.Source, err)
			}
		}
		t, err = t.New(item.Path).Parse(string(item.Content.Data))
		if err != nil {
			return nil, fmt.Errorf("unable to parse template file %s from %s: %w", item.Path, item.Source, err)
		}
	}
	return t, nil
}

// TemplateFilenames executes a template on a filename and returns the result.
// This allows dynamic filename generation based on template data.
func TemplateFilenames(filenameTemplate *template.Template, outputPath string, mergedData map[string]any) (string, error) {
	_, err := filenameTemplate.New(outputPath).Parse(outputPath)
	if err != nil {
		return "", fmt.Errorf("error parsing filename template: %w", err)
	}
	templatedPath := new(bytes.Buffer)
	err = filenameTemplate.ExecuteTemplate(templatedPath, outputPath, mergedData)
	if err != nil {
		templateErr := &types.TemplateError{
			TemplatePath: outputPath,
			Err:          err,
		}
		return "", templateErr
	}
	return templatedPath.String(), nil
}

func InitTemplate(sharedTemplates []*data.FileArg, strict bool) (*template.Template, error) {
	// Create ONE template for everything (like Helm does)
	var onError string
	if strict {
		onError = "error"
	} else {
		onError = "zero"
	}

	t := template.New("yutc").Option("missingkey=" + onError)

	sprigFuncMap := sprig.TxtFuncMap()

	// Add custom functions to the map
	customFuncMap := GetCustomFuncMap()

	// Add include/tpl functions
	includedNames := make(map[string]int)
	helmLikeFuncMap := template.FuncMap{
		"include": IncludeFun(t, includedNames),
		"tpl":     TplFun(t, includedNames, strict),
	}

	// Load all function before parsing
	t = t.Funcs(sprigFuncMap).Funcs(helmLikeFuncMap).Funcs(customFuncMap)

	// Parse shared templates
	for idx, sharedTemplateBuffer := range sharedTemplates {
		sharedName := "shared-" + strconv.Itoa(idx)
		// It is assumed that shared templates will primarily contain 'define' blocks
		// which are then referenced by their defined name using 'include'.
		// The sharedName here is really only for debugging purposes at this time

		_, err := t.New(sharedName).Parse(sharedTemplateBuffer.String())
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

// GetCustomFuncMap returns only the custom yutc functions (no Sprig, no include/tpl).
func GetCustomFuncMap() template.FuncMap {
	return template.FuncMap{
		"toYaml":       ToYaml,
		"fromYaml":     FromYaml,
		"mustToYaml":   MustToYaml,
		"yamlOptions":  SetYamlEncodeOptions,
		"mustFromYaml": MustFromYaml,
		"toToml":       ToToml,
		"fromToml":     FromToml,
		"mustToToml":   MustToToml,
		"mustFromToml": MustFromToml,
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
		"sortKeys":     SortKeys,
		"sortList":     SortList,
		"shellQuote":   quote.ShellQuote,
		"luaQuote":     quote.LuaQuote,
	}
}
