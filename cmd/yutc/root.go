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

func newRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "yutc",
		Short: "Yet Unnamed Template CLI",
		Args:  cobra.MinimumNArgs(0),
		RunE:  runRoot,
	}
}

func runRoot(cmd *cobra.Command, args []string) (err error) {
	runSettings.TemplatePaths = args
	if YutcLog.GetLevel() < 0 {
		logSettings()
	}

	if runSettings.Version {
		internal.PrintVersion()
		return nil
	}

	if len(runSettings.TemplatePaths) == 0 {
		YutcLog.Fatal().Msg("No template files specified")
	}

	// Recursive and apply filters to inputs as necessary
	templateFiles, _ := resolvePaths(runSettings.TemplatePaths, tempDir)
	YutcLog.Debug().Msg(fmt.Sprintf("Found %d template files", len(templateFiles)))
	for _, templateFile := range templateFiles {
		YutcLog.Trace().Msg("  - " + templateFile)
	}

	dataFiles, _ := resolvePaths(runSettings.DataFiles, tempDir)
	YutcLog.Debug().Msg(fmt.Sprintf("Found %d data files", len(dataFiles)))
	for _, dataFile := range dataFiles {
		YutcLog.Trace().Msg("  - " + dataFile)
	}

	commonFiles, _ := resolvePaths(runSettings.CommonTemplateFiles, tempDir)
	YutcLog.Debug().Msg(fmt.Sprintf("Found %d common template files", len(commonFiles)))
	for _, commonFile := range commonFiles {
		YutcLog.Trace().Msg("  - " + commonFile)
	}
	exitCode, errs := internal.ValidateArguments(runSettings)
	internal.ExitCode = &exitCode
	if *internal.ExitCode > 0 {
		var errStrings []string
		for _, err := range errs {
			errStrings = append(errStrings, err.Error())
		}
		return fmt.Errorf("validation errors: %v", errStrings)
	}

	data, err := internal.MergeData(dataFiles)
	if err != nil {
		panic(err)
	}
	commonTemplates := internal.LoadSharedTemplates(runSettings.CommonTemplateFiles)
	templates, err := internal.LoadTemplates(templateFiles, commonTemplates)
	if err != nil {
		YutcLog.Panic().Msg(err.Error())
	}
	for templateIndex, tmpl := range templates {
		var outData *bytes.Buffer
		outData = new(bytes.Buffer)
		err = tmpl.Execute(outData, data)
		if err != nil {
			YutcLog.Panic().Msg(err.Error())
		}
		outputOriginalPath := internal.NormalizeFilepath(runSettings.Output)
		templateOriginalPath := templateFiles[templateIndex] // as the user provided
		outputBasename := filepath.Base(templateOriginalPath)
		// stdin isn't handled here, gotta do that
		if runSettings.Output != "-" {
			// execute filenames as templates if requested
			if runSettings.IncludeFilenames {
				templateOriginalPath = templateFilenames(templateOriginalPath, commonTemplates, data)
			}

			inputIsDir, _ := internal.CheckIfDir(outputOriginalPath)                        // err already checked in validation
			_ = resolveFileOutput(outputOriginalPath, templateOriginalPath, inputIsDir, "") // TESTING

			if inputIsDir {
				err = os.MkdirAll(templateOriginalPath, 0755)
				if err != nil {
					panic(err)
				}
				// no other work needed since it's just a directory, moving on
				continue
			}

			var outputPath string
			if len(templates) > 1 {
				outputPath = internal.NormalizeFilepath(filepath.Join(runSettings.Output, outputBasename))
			} else {
				outputPath = internal.NormalizeFilepath(runSettings.Output)
			}

			_ = filepath.Dir(outputPath)
			outputBasename = filepath.Base(outputPath)

			isDir, err := internal.CheckIfDir(outputPath)
			if err == nil && isDir && len(templates) == 1 {
				// behavior for single template file and output is a directory
				// matches normal behavior expected by commands like cp, mv etc.
				outputPath = filepath.Join(runSettings.Output, outputBasename)
				isDir, err = internal.CheckIfDir(outputPath)
				if err != nil {
					YutcLog.Fatal().Msg(err.Error())
				}
			}

			// check again in case the output path was changed and the file still exists,
			// we can probably make this into just one case statement but it's late and i am tired
			if runSettings.IncludeFilenames {
				outputPath = templateFilenames(outputPath, commonTemplates, data)
			}
			isDir, err = internal.CheckIfDir(outputPath)
			// the error here is going to be that the file doesn't exist
			if err != nil || (!isDir && runSettings.Overwrite) {
				if runSettings.Overwrite {
					YutcLog.Debug().Msg("Overwrite enabled, writing to file(s): " + runSettings.Output)
				}
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
	return err
}

func checkRetainFolderStructure(paths []string) bool {
	if len(paths) != 1 {
		return false
	}
	isDir, err := internal.CheckIfDir(paths[0])
	if err != nil {
		YutcLog.Panic().Msg(
			"We should never get to this point, this should have been checked during arg validation",
		)
	}
	if isDir {
		return true
	}
	return internal.IsArchive(paths[0])
}

func resolveFileOutput(outputPath, inputPath string, outputPathIsDir bool, nestedBase string) string {
	isFile, err := internal.CheckIfFile(outputPath)
	if isFile && err == nil {
		return outputPath // one output specified, no need to change it
	} else if err != nil {

		inputIsDir, err := internal.CheckIfDir(outputPath) // does the target exist and is it a directory?
		if err == nil {
			if inputIsDir {
				// we need to add the basename of the input file/path to it (like in cp)
				updatedOutputPath := filepath.Join(outputPath, filepath.Base(inputPath))
				outputPath = resolveFileOutput(updatedOutputPath, inputPath, inputIsDir, nestedBase)
			}
		}
		return outputPath
	}
	return "whatever"
}

func logSettings() {
	YutcLog.Trace().Msg("Settings:")
	yamlSettings, err := yaml.Marshal(runSettings)
	if err != nil {
		panic(err) // this should never happen unless we seriously goofed up
	}
	for _, line := range bytes.Split(yamlSettings, []byte("\n")) {
		YutcLog.Trace().Msg("  " + string(line))
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
func resolvePaths(paths []string, tempDir string) ([]string, error) {
	var outFiles []string
	var filename string
	var data []byte
	recursables, err := internal.CountRecursables(paths)
	if err != nil {
		return nil, err
	}

	if recursables > 0 {
		for _, templatePath := range paths {
			source, err := internal.ParseFileStringFlag(templatePath)
			if err != nil {
				panic(err)
			}
			switch source {
			case "stdin":
			case "url":
				filename, data, _, err = internal.ReadUrl(templatePath)
				tempPath := filepath.Join(tempDir, filename)
				if err != nil {
					return nil, err
				}
				err = os.WriteFile(tempPath, data, 0644)
				if err != nil {
					return nil, err
				}
				templatePath = tempPath
				fallthrough
			default:
				templatePath = filepath.ToSlash(templatePath)
				filteredPaths := internal.WalkDir(templatePath)
				outFiles = append(outFiles, filteredPaths...)
			}
		}
	} else {
		for _, templatePath := range paths {
			source, err := internal.ParseFileStringFlag(templatePath)
			if err != nil {
				panic(err)
			}
			if source == "url" {
				filename, data, _, err := internal.ReadUrl(templatePath)
				tempPath := filepath.Join(tempDir, filename)
				if err != nil {
					YutcLog.Fatal().Msg(err.Error())
				}
				errRaw := os.WriteFile(tempPath, data, 0644)
				if errRaw != nil {
					return nil, errRaw
				}
				templatePath = tempPath
			}
			outFiles = append(outFiles, templatePath)
		}
	}

	return outFiles, nil
}
