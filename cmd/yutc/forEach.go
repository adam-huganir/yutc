package main

import (
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path"
	"slices"
)

func runForEachCommand(cmd *cobra.Command, args []string) (err error) {
	YutcLog.Trace().Msg("yutc.runForEachCommand() called")

	runSettings.TemplatePaths = args

	err = parseArgs(runSettings)
	if err != nil {
		return err
	}

	var datas []any
	if len(runSettings.DataFiles) == 1 {
		data, _, err := internal.GatherData(runSettings.DataFiles, false, runSettings.BasicAuth, runSettings.BearerToken)
		if err != nil {
			return err
		}
		datas = data.([]any)
	} else {
		for _, dataFile := range runSettings.DataFiles {
			data, _, err := internal.GatherData([]string{dataFile}, false, runSettings.BasicAuth, runSettings.BearerToken)
			if err != nil {
				return err
			}
			datas = append(datas, data)
		}
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
	first := true
	for _, t := range templates {
		t.SetRelativePath(templateRootDir)
		for _, data := range datas {
			if runSettings.Output == "-" {
				outWriter = os.Stdout
				if !first {
					_, _ = outWriter.Write([]byte("---\n"))
				} else {
					first = false
				}
			} else {
				outWriter, err = createWriter(t.RelativePath(), runSettings.Output, runSettings.Overwrite)
				if err != nil {
					return err
				}
			}
			_, err = evalTemplate(t, commonTemplates, data, outWriter)
		}
	}

	return nil
}
