package main

import (
	"bytes"
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var runSettings = &internal.CLISettings{}

var rootCommand = &cobra.Command{
	Use:   "yutc",
	Short: "Yet Unnamed Template CLI",
	Args:  cobra.MinimumNArgs(1),
	Run:   runRoot,
}

func runRoot(cmd *cobra.Command, args []string) {
	var err error

	// Define flags
	//var err error
	//
	//_, err = internal.ReadTar("eg.tgz")
	//if err != nil {
	//	panic(err)
	//}

	if err != nil {
		YutcLog.Error().Msg(err.Error())
		//os.Exit(10)
	}
	runSettings.TemplateFiles = cmd.Flags().Args()
	if runSettings.Version {
		internal.PrintVersion()
		os.Exit(0)
	}

	if YutcLog.GetLevel() < 0 {
		YutcLog.Trace().Msg("Settings:")
		yamlSettings, err := yaml.Marshal(runSettings)
		if err != nil {
			panic(err) // this should never happen unless we goofed up
		}
		for _, line := range bytes.Split(yamlSettings, []byte("\n")) {
			YutcLog.Trace().Msg("  " + string(line))
		}
	}

	valCode := internal.ValidateArguments(runSettings)
	if valCode > 0 {
		YutcLog.Error().Msg("Invalid arguments")
		os.Exit(int(valCode))
	}

	var templateFiles []string
	if runSettings.Recursive {
		for _, templateFile := range runSettings.TemplateFiles {
			rootDirFS := os.DirFS(templateFile)
			filteredPaths := internal.WalkDir(rootDirFS, runSettings.IncludePatterns, runSettings.ExcludePatterns)
			templateFiles = append(templateFiles, filteredPaths...)
		}
	} else {
		templateFiles = runSettings.TemplateFiles
	}

	data, err := internal.MergeData(runSettings.DataFiles)
	if err != nil {
		panic(err)
	}

	commonTemplates := internal.LoadSharedTemplates(runSettings.CommonTemplateFiles)

	templates, err := internal.LoadTemplates(templateFiles, commonTemplates)
	for templateIndex, tmpl := range templates {
		var outData *bytes.Buffer
		outData = new(bytes.Buffer)
		err = tmpl.Execute(outData, data)
		if err != nil {
			panic(err)
		}
		basename := filepath.Base(templateFiles[templateIndex])
		// stdin not handled here, gotta do that
		if runSettings.Output != "-" {
			var outputPath string
			if len(templates) > 1 {
				outputPath = filepath.Join(runSettings.Output, basename)
			} else {
				outputPath = runSettings.Output
			}
			if err != nil {
				panic(err)
			}
			isDir, err := checkIfDir(outputPath)
			if err == nil && *isDir && len(templates) == 1 {
				// behavior for single template file and output is a directory
				// matches normal behavior expected by commands like cp, mv etc.
				outputPath = filepath.Join(runSettings.Output, basename)
			}
			// check again in case the output path was changed and the file still exists,
			// we can probably make this into just one case statement but it's late and i am tired
			isDir, err = checkIfDir(outputPath)
			// error here is going to be that the file doesnt exist
			if err != nil || (!*isDir && runSettings.Overwrite) {
				YutcLog.Debug().Msg("Overwrite enabled, writing to file(s): " + runSettings.Output)
				err = os.WriteFile(outputPath, outData.Bytes(), 0644)
				if err != nil {
					panic(err)
				}
			} else {
				YutcLog.Error().Msg("file exists and overwrite is not set: " + outputPath)
			}
		} else {
			YutcLog.Debug().Msg("Writing to stdout")
			_, err = os.Stdout.Write(outData.Bytes())
			if err != nil {
				panic(err)
			}
		}
	}
}
