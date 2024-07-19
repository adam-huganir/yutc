package main

import (
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path"
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

	slices.SortFunc(templates, internal.CmpTemplatePathLength)                   // sort templates by their file path length (shortest first)
	templateRootDir := internal.NormalizeFilepath(path.Dir(templates[0].Path())) // because of above we know this is the root dir

	// see if we need to dive into these and pull files out of them

	var outWriter io.Writer

	// Load up our output templates with any common definitions from the shared templates
	for _, t := range templates {
		t.SetRelativePath(templateRootDir)
		if runSettings.Output == "-" {
			outWriter = os.Stdout
		} else {
			outWriter, err = createWriter(t.RelativePath(), runSettings.Output, runSettings.Overwrite)
			if err != nil {
				return err
			}
		}
		_, err = evalTemplate(t, commonTemplates, data, outWriter)
	}

	return nil
}
