package yutc

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/adam-huganir/yutc/pkg/config"
	"github.com/adam-huganir/yutc/pkg/data"
	"github.com/adam-huganir/yutc/pkg/files"
	templatePkg "github.com/adam-huganir/yutc/pkg/template"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/goccy/go-yaml"
	"github.com/rs/zerolog"
)

type App struct {
	Settings *types.YutcSettings
	Data     *config.RunData
	Logger   zerolog.Logger
}

func NewApp(settings *types.YutcSettings, logger zerolog.Logger) *App {
	return &App{
		Settings: settings,
		Data: &config.RunData{
			YutcSettings: settings,
		},
		Logger: logger,
	}
}

func (app *App) Run(args []string) (err error) {
	app.Settings.TemplatePaths = args
	if app.Logger.GetLevel() < 0 {
		app.LogSettings()
	}

	if app.Settings.Version {
		PrintVersion()
		return nil
	}

	if len(app.Settings.TemplatePaths) == 0 {
		app.Logger.Fatal().Msg("No template files specified")
	}

	// Recursive and apply filters to inputs as necessary
	tempDir, _ := files.GenerateTempDirName("yutc-*")
	// defer os.RemoveAll(tempDir) // TODO: Decide if we want to clean this up

	templateFiles, _ := ResolvePaths(app.Settings.TemplatePaths, tempDir, app.Logger)
	// this sort will help us later when we make assumptions about if folders already exist
	slices.SortFunc(templateFiles, func(a, b string) int {
		aIsShorter := len(a) < len(b)
		if aIsShorter {
			return -1
		}
		return 1
	})

	app.Logger.Debug().Msg(fmt.Sprintf("Found %d template files", len(templateFiles)))
	for _, templateFile := range templateFiles {
		app.Logger.Trace().Msg("  - " + templateFile)
	}

	err = app.Data.ParseDataFiles()
	if err != nil {
		return err
	}
	dataFiles, err := ResolveDataPaths(app.Data.DataFiles, tempDir, app.Logger)
	if err != nil {
		return err
	}
	app.Logger.Debug().Msg(fmt.Sprintf("Found %d data files", len(dataFiles)))
	for _, dataFile := range dataFiles {
		app.Logger.Trace().Msg("  - " + dataFile.Path)
	}

	commonFiles, _ := ResolvePaths(app.Settings.CommonTemplateFiles, tempDir, app.Logger)
	app.Logger.Debug().Msgf("Found %d common template files", len(commonFiles))
	for _, commonFile := range commonFiles {
		app.Logger.Trace().Msg("  - " + commonFile)
	}
	exitCode, errs := config.ValidateArguments(app.Settings, app.Logger)
	if exitCode > 0 {
		var errStrings []string
		for _, err := range errs {
			errStrings = append(errStrings, err.Error())
		}
		return &types.ExitError{Code: exitCode, Err: fmt.Errorf("validation errors: %v", errStrings)}
	}

	mergedData, err := data.MergeData(dataFiles, app.Logger)
	if err != nil {
		panic(err)
	}
	commonTemplates := data.LoadSharedTemplates(app.Settings.CommonTemplateFiles, app.Logger)
	templates, err := templatePkg.LoadTemplates(templateFiles, commonTemplates, app.Settings.Strict, app.Logger)
	if err != nil {
		app.Logger.Panic().Msg(err.Error())
	}

	// we rely on validation to make sure we aren't getting multiple recursables
	firstTemplatePath := templateFiles[0]
	inputIsRecursive, err := files.IsDir(firstTemplatePath)
	if !inputIsRecursive {
		inputIsRecursive = files.IsArchive(firstTemplatePath)
	}
	resolveRoot := ""
	if err == nil && inputIsRecursive {
		resolveRoot = firstTemplatePath
	}
	for templateIndex, tmpl := range templates {
		templateOriginalPath := templateFiles[templateIndex] // as the user provided

		// if we have a directory as our template source we want to keep track of relative paths
		// execute filenames as templates if requested
		var relativePath string
		var templateOutputPath = templateOriginalPath
		if app.Settings.IncludeFilenames {
			templateOutputPath = TemplateFilenames(templateOriginalPath, commonTemplates, mergedData, app.Settings.Strict, app.Logger)
		}
		if inputIsRecursive {
			relativePath = ResolveFileOutput(templateOutputPath, resolveRoot) // TESTING
		} else if err == nil { // i.e. it's a file
			relativePath = path.Base(templateOutputPath)
		}

		var outputPath string
		if app.Settings.Output != "-" {
			outputIsDir, _ := files.IsDir(templateOriginalPath)
			if outputIsDir {
				outputPath = files.NormalizeFilepath(filepath.Join(app.Settings.Output, relativePath))
				err = os.MkdirAll(outputPath, 0755)
				if err != nil {
					panic(err)
				}
				// no other work needed since it's just a directory, moving on
				continue
			} else {
				if inputIsRecursive {
					outputPath = files.NormalizeFilepath(filepath.Join(app.Settings.Output, relativePath))
				} else {
					outputPath = files.NormalizeFilepath(filepath.Join(app.Settings.Output))
				}
			}
			if tmpl == nil {
				panic("template is nil, this should never happen but haven't fully tested this yet to be sure")
			}
		}
		outData := new(bytes.Buffer)
		err = tmpl.Execute(outData, mergedData)
		if err != nil {
			app.Logger.Panic().Msg(err.Error())
		}
		if app.Settings.Output == "-" {
			app.Logger.Debug().Msg("Writing to stdout")
			_, err = os.Stdout.Write(outData.Bytes())
			if err != nil {
				panic(err)
			}
		} else {

			_ = filepath.Dir(outputPath)
			outputBasename := filepath.Base(outputPath)

			isDir, err := files.IsDir(outputPath)
			if err == nil && isDir && len(templates) == 1 {
				// behavior for single template file and output is a directory
				// matches normal behavior expected by commands like cp, mv etc.
				outputPath = filepath.Join(app.Settings.Output, outputBasename)
				_, err = files.IsDir(outputPath)
				if err != nil {
					app.Logger.Fatal().Msg(err.Error())
				}
			}

			// check again in case the output path was changed and the file still exists,
			// we can probably make this into just one case statement but it's late and i am tired
			if app.Settings.IncludeFilenames {
				outputPath = TemplateFilenames(outputPath, commonTemplates, mergedData, app.Settings.Strict, app.Logger)
			}
			isDir, err = files.IsDir(outputPath)
			// the error here is going to be that the file doesn't exist
			if err != nil || (!isDir && app.Settings.Overwrite) {
				if app.Settings.Overwrite {
					app.Logger.Debug().Msg("Overwrite enabled, writing to file(s): " + app.Settings.Output)
				}
				err = os.WriteFile(outputPath, outData.Bytes(), 0644)
				if err != nil {
					panic(err)
				}
			} else {
				app.Logger.Error().Msg("file exists and overwrite is not set: " + outputPath)
			}
		}
	}
	return err
}

func ResolveFileOutput(inputPath, nestedBase string) string {
	if nestedBase == "" {
		return inputPath
	}
	if inputPath == nestedBase {
		return "."
	}
	if nestedBase[len(nestedBase)-1] != '/' {
		nestedBase += "/" // ensure we have a trailing slash so we can remove it from the input path
	}
	return strings.TrimPrefix(inputPath, nestedBase)
}

func (app *App) LogSettings() {
	app.Logger.Trace().Msg("Settings:")
	yamlSettings, err := yaml.Marshal(app.Settings)
	if err != nil {
		panic(err) // this should never happen unless we seriously goofed up
	}
	for _, line := range bytes.Split(yamlSettings, []byte("\n")) {
		app.Logger.Trace().Msg("  " + string(line))
	}
}

func TemplateFilenames(outputPath string, commonTemplates []*bytes.Buffer, data map[string]any, strict bool, logger zerolog.Logger) string {
	filenameTemplate, err := templatePkg.BuildTemplate(outputPath, commonTemplates, "filename", strict)
	if err != nil {
		logger.Fatal().Msg(err.Error())
		return ""
	}
	if filenameTemplate == nil {
		err = fmt.Errorf("error building filename template for %s", outputPath)
		logger.Fatal().Msg(err.Error())
		return ""
	}
	templatedPath := new(bytes.Buffer)
	err = filenameTemplate.Execute(templatedPath, data)
	if err != nil {
		logger.Fatal().Msg(err.Error())
		return ""
	}
	return templatedPath.String()
}

// Introspect each template and resolve to a file, or if it is a path to a directory,
// resolve all files in that directory.
// After applying the specified match/exclude patterns, return the list of files.
func ResolvePaths(paths []string, tempDir string, logger zerolog.Logger) ([]string, error) {
	var outFiles []string
	var filename string
	var data []byte
	recursables, err := files.CountRecursables(paths)
	if err != nil {
		return nil, err
	}

	if recursables > 0 {
		for _, templatePath := range paths {
			source, err := files.ParseFileStringFlag(templatePath)
			if err != nil {
				panic(err)
			}
			switch source {
			case "stdin":
			case "url":
				filename, data, _, err = files.ReadUrl(templatePath, logger)
				tempPath := filepath.Join(tempDir, filename)
				if err != nil {
					return nil, err
				}
				tempDirExists, _ := files.Exists(tempPath)
				if !tempDirExists {
					err = os.Mkdir(tempPath, 0755)
					if err != nil {
						logger.Panic().Msg(err.Error())
					}
				}
				err = os.WriteFile(tempPath, data, 0644)
				if err != nil {
					return nil, err
				}
				templatePath = tempPath
				fallthrough
			default:
				templatePath = filepath.ToSlash(templatePath)
				filteredPaths := files.WalkDir(templatePath, logger)
				outFiles = append(outFiles, filteredPaths...)
			}
		}
	} else {
		for _, templatePath := range paths {
			source, err := files.ParseFileStringFlag(templatePath)
			if err != nil {
				panic(err)
			}
			if source == "url" {
				filename, data, _, err := files.ReadUrl(templatePath, logger)
				tempPath := filepath.Join(tempDir, filename)
				if err != nil {
					logger.Fatal().Msg(err.Error())
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

func ResolveDataPaths(dataFiles []*types.DataFileArg, tempDir string, logger zerolog.Logger) ([]*types.DataFileArg, error) {
	dataPathsOnly := make([]string, len(dataFiles))
	for idx, dataFile := range dataFiles {
		dataPathsOnly[idx] = dataFile.Path
	}
	paths, err := ResolvePaths(dataPathsOnly, tempDir, logger)
	if err != nil {
		return nil, err
	}
	for idx, newPath := range paths {
		dataFiles[idx].Path = newPath
	}
	return dataFiles, nil
}
