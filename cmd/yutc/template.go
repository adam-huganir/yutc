package main

import (
	"github.com/adam-huganir/yutc/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"slices"
)

func runTemplateCommand(cmd *cobra.Command, args []string) (err error) {
	YutcLog.Trace().Msg("yutc.runTemplateCommand() called")

	runSettings.TemplatePaths = args

	err = parseArgs(runSettings)
	if err != nil {
		return err
	}

	data, _, err := internal.GatherData(runSettings.DataFiles, false, runSettings.BasicAuth, runSettings.BearerToken)
	if err != nil {
		return err
	}
	var commonTemplates []*internal.FileData
	commonTemplates, err = internal.LoadFiles(runSettings.CommonTemplateFiles, runSettings.BasicAuth, runSettings.BearerToken)
	if err != nil {
		return err
	}

	var templates []*internal.YutcTemplate
	templates, err = internal.LoadTemplates(runSettings.TemplatePaths, runSettings.BasicAuth, runSettings.BearerToken)
	if err != nil {
		return err
	}

	slices.SortFunc(templates, internal.CmpTemplatePathLength) // sort templates by their file path length (shortest first)
	templateRootPath := internal.NormalizeFilepath(templates[0].Path())

	// see if we need to dive into these and pull files out of them

	var outWriter io.Writer

	// Load up our output templates with any common definitions from the shared templates
	for _, t := range templates {
		t.SetRelativePath(templateRootPath)
		if runSettings.Output == "-" {
			outWriter = os.Stdout
		} else {
			outPath := filepath.Join(runSettings.Output, t.RelativePath())
			outDir := filepath.Dir(outPath)
			exists, err := internal.Exists(outDir)
			if !exists && err == nil {
				err = internal.Fs.Mkdir(outDir, 0755)
				if err != nil {
					return err
				}
			}
			outWriter, err = internal.Fs.Create(outPath)
			if err != nil {
				return err
			}
		}

		for _, ct := range commonTemplates {
			err = t.AddTemplate(ct.ReadWriter.String())
			if err != nil {
				return err
			}
		}
		result, err := t.Execute(data)
		if err != nil {
			return errors.Wrapf(err, "error executing template %s", t.ID())
		}
		err = nil
		for err == nil {
			_, err = outWriter.Write(result.Bytes())
		}
		println("---")
	}

	return nil
}
