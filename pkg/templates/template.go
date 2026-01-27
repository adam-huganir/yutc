package templates

import (
	"fmt"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/adam-huganir/yutc/pkg/data"
	"github.com/adam-huganir/yutc/pkg/quote"
	"github.com/rs/zerolog"
)

// TemplateSet holds a single template with all parsed templates and their source information.
type TemplateSet struct {
	Template      *template.Template
	TemplateFiles []*data.FileArg
}

// LoadTemplateSet loads template data and parses them with shared templates and custom functions.
// Following Helm's approach: creates ONE template object, parses all data into it.
func LoadTemplateSet(
	templateFiles []*data.FileArg,
	sharedTemplateBuffers []*data.FileArg,
	mergedData map[string]any,
	strict, includeFilenames bool,
	logger *zerolog.Logger,
) (*TemplateSet, error) {
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
		logger.Debug().Msgf("Loading from %s template file %s", templateFile.Source, templateFile.Name)
	}
	if includeFilenames {
		filenameTemplate, err := InitTemplate(sharedTemplateBuffers, strict)
		if err != nil {
			return nil, fmt.Errorf("error initializing filename template: %w", err)
		}
		err = data.TemplateFilenames(templateItems, filenameTemplate, mergedData)
		if err != nil {
			return nil, err
		}
	}

	t, err = ParseTemplateItems(t, templateItems)
	if err != nil {
		return nil, err
	}
	return &TemplateSet{
		Template:      t,
		TemplateFiles: templateItems,
	}, nil
}

// ParseTemplateItems parses template data into the same template object.
func ParseTemplateItems(t *template.Template, items []*data.FileArg) (*template.Template, error) {
	var err error
	for _, item := range items {
		if !item.Content.Read {
			err = item.Load()
			if err != nil {
				return nil, fmt.Errorf("unable to load template file %s from %s: %w", item.Name, item.Source, err)
			}
		}
		name := item.Name
		if item.NewName != "" {
			name = item.NewName
		}
		t, err = t.New(name).Parse(string(item.Content.Data))
		if err != nil {
			return nil, fmt.Errorf("unable to parse template file %s from %s: %w", item.Name, item.Source, err)
		}
	}
	return t, nil
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
		if !sharedTemplateBuffer.Content.Read {
			err := sharedTemplateBuffer.Load()
			if err != nil {
				return nil, err
			}
		}
		_, err := t.New(sharedName).Parse(string(sharedTemplateBuffer.Content.Data))
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
