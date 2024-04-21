package main

import (
	"bytes"
	"fmt"
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var rootCommand = &cobra.Command{
	Use:   "yutc",
	Short: "Yet Unnamed Template CLI",
	Args:  cobra.MinimumNArgs(0),
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

	//if err != nil {
	//	YutcLog.Error().Msg(err.Error())
	//	//os.Exit(10)
	//}
	internal.RunSettings.TemplatePaths = cmd.Flags().Args()
	if len(internal.RunSettings.TemplatePaths) == 0 {
	}
	if internal.RunSettings.Version {
		internal.PrintVersion()
		os.Exit(0)
	}

	if YutcLog.GetLevel() < 0 {
		YutcLog.Trace().Msg("Settings:")
		yamlSettings, err := yaml.Marshal(internal.RunSettings)
		if err != nil {
			panic(err) // this should never happen unless we goofed up
		}
		for _, line := range bytes.Split(yamlSettings, []byte("\n")) {
			YutcLog.Trace().Msg("  " + string(line))
		}
	}

	// Recursive and apply filters as necessary
	templateFiles := resolvePaths(internal.RunSettings.TemplatePaths, internal.RunSettings.TemplateMatch)
	YutcLog.Debug().Msg(fmt.Sprintf("Found %d template files", len(templateFiles)))
	for _, templateFile := range templateFiles {
		YutcLog.Trace().Msg("  - " + templateFile)
	}
	dataFiles := resolvePaths(internal.RunSettings.DataFiles, internal.RunSettings.DataMatch)
	YutcLog.Debug().Msg(fmt.Sprintf("Found %d data files", len(dataFiles)))
	for _, dataFile := range dataFiles {
		YutcLog.Trace().Msg("  - " + dataFile)

	}
	commonFiles := resolvePaths(internal.RunSettings.CommonTemplateFiles, internal.RunSettings.CommonTemplateMatch)
	YutcLog.Debug().Msg(fmt.Sprintf("Found %d common template files", len(commonFiles)))
	for _, commonFile := range commonFiles {
		YutcLog.Trace().Msg("  - " + commonFile)
	}

	valCode := internal.ValidateArguments(internal.RunSettings)
	if valCode > 0 {
		YutcLog.Error().Msg("Invalid arguments")
		os.Exit(int(valCode))
	}

	data, err := internal.MergeData(dataFiles)
	if err != nil {
		panic(err)
	}
	commonTemplates := internal.LoadSharedTemplates(internal.RunSettings.CommonTemplateFiles)

	templates, err := internal.LoadTemplates(templateFiles, commonTemplates)
	if err != nil {
		panic(err)
	}
	for templateIndex, tmpl := range templates {
		var outData *bytes.Buffer
		outData = new(bytes.Buffer)
		err = tmpl.Execute(outData, data)
		if err != nil {
			panic(err)
		}
		basename := filepath.Base(templateFiles[templateIndex])
		// stdin isn't handled here, gotta do that
		if internal.RunSettings.Output != "-" {
			var outputPath string
			if len(templates) > 1 {
				outputPath = filepath.Join(internal.RunSettings.Output, basename)
			} else {
				outputPath = internal.RunSettings.Output
			}

			// execute filenames as templates if requested
			if internal.RunSettings.IncludeFilenames {
				outputPath = templateFilenames(outputPath, commonTemplates, data)
			}
			basename = filepath.Base(outputPath)

			isDir, err := internal.CheckIfDir(outputPath)
			if err == nil && *isDir && len(templates) == 1 {
				// behavior for single template file and output is a directory
				// matches normal behavior expected by commands like cp, mv etc.
				outputPath = filepath.Join(internal.RunSettings.Output, basename)
				isDir, err = internal.CheckIfDir(outputPath)
				if err != nil {
					YutcLog.Fatal().Msg(err.Error())
				}
			}

			// check again in case the output path was changed and the file still exists,
			// we can probably make this into just one case statement but it's late and i am tired
			if internal.RunSettings.IncludeFilenames {
				outputPath = templateFilenames(outputPath, commonTemplates, data)
			}
			isDir, err = internal.CheckIfDir(outputPath)
			// the error here is going to be that the file doesn't exist
			if err != nil || (!*isDir && internal.RunSettings.Overwrite) {
				YutcLog.Debug().Msg("Overwrite enabled, writing to file(s): " + internal.RunSettings.Output)

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

func templateFilenames(outputPath string, commonTemplates []*bytes.Buffer, data map[string]any) string {
	filenameTemplate, err := internal.BuildTemplate(outputPath, commonTemplates)
	if err != nil {
		YutcLog.Fatal().Msg(err.Error())
		return ""
	}
	if filenameTemplate == nil {
		err = fmt.Errorf("error building filename template for %s", outputPath)
		YutcLog.Fatal().Msg(err.Error())
		return ""
	}
	templatedPath := new(bytes.Buffer)
	err = filenameTemplate.Execute(templatedPath, data)
	if err != nil {
		YutcLog.Fatal().Msg(err.Error())
		return ""
	}
	return templatedPath.String()
}

// Introspect each template and resolve to a file, or if it is a path to a directory,
// resolve all files in that directory.
// After applying the specified match/exclude patterns, return the list of files.
func resolvePaths(paths, matches []string) []string {
	var outFiles []string
	recursables := 0
	for _, templateFile := range paths {
		isDir, err := internal.CheckIfDir(templateFile)
		if err != nil {
			YutcLog.Fatal().Msg(err.Error())
		}
		if *isDir || internal.IsArchive(templateFile) {
			recursables++
		}
	}
	if matches != nil {
		if recursables > 0 {
			for _, templatePath := range paths {
				source, err := internal.ParseFileStringFlag(templatePath)
				if err != nil {
					panic(err)
				}
				switch source {
				case "url", "stdin":
					outFiles = append(outFiles, templatePath)
				default:
					templatePath = filepath.ToSlash(templatePath)
					filteredPaths := internal.WalkDir(templatePath, matches)
					outFiles = append(outFiles, filteredPaths...)
				}
			}
		} else {
			YutcLog.Fatal().Msg("Match/exclude patterns are not supported for single files")
		}
	} else {
		outFiles = paths
	}

	return outFiles
}
