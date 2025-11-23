// Package yutc provides the core application logic for the yutc template processor.
package yutc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/adam-huganir/yutc/pkg/config"
	"github.com/adam-huganir/yutc/pkg/data"
	"github.com/adam-huganir/yutc/pkg/files"
	yutcTemplate "github.com/adam-huganir/yutc/pkg/template"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/goccy/go-yaml"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/theory/jsonpath"
)

// App holds the application state and dependencies for template execution.
type App struct {
	Settings *types.Arguments
	RunData  *types.RunData
	Logger   *zerolog.Logger
	Command  *cobra.Command
	TempDir  string
}

// NewApp creates a new App instance with the provided settings, data, logger, and command.
func NewApp(settings *types.Arguments, runData *types.RunData, logger *zerolog.Logger, cmd *cobra.Command) *App {
	tempDir, _ := files.GenerateTempDirName("yutc-*")
	return &App{
		Settings: settings,
		RunData:  runData,
		Logger:   logger,
		Command:  cmd,
		TempDir:  tempDir,
	}
}

// Run executes the yutc application with the provided context and template arguments.
// It loads data files, parses templates, and generates output based on the configured settings.
func (app *App) Run(_ context.Context, args []string) (err error) {
	app.Settings.TemplatePaths = args
	if app.Logger.GetLevel() < zerolog.DebugLevel {
		app.LogSettings()
	}

	if app.Settings.Version {
		PrintVersion()
		return nil
	}

	if len(app.Settings.TemplatePaths) == 0 {
		app.Logger.Fatal().Msg("No template files specified")
	}

	// grab the name of a temp directory to use for processing
	// but it is not guaranteed to exist yet
	tempDir := app.TempDir
	defer func() {
		if exists, _ := files.Exists(tempDir); exists {
			_ = os.RemoveAll(tempDir)
		}
	}()

	templateFiles, err := data.LoadTemplates(app.Settings.TemplatePaths, tempDir, app.Logger)
	if err != nil {
		return err
	}

	err = data.ParseDataFiles(app.RunData, app.Settings.DataFiles)
	if err != nil {
		return err
	}
	dataFiles, err := data.LoadDataFiles(app.RunData.DataFiles, tempDir, app.Logger)
	if err != nil {
		return err
	}

	commonFiles, _ := files.ResolvePaths(app.Settings.CommonTemplateFiles, tempDir, app.Logger)
	err = data.ParseTemplatePaths(app.RunData, commonFiles)
	if err != nil {
		return err
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
		return err
	}
	// parse our explicitly set values
	for _, ss := range app.Settings.SetData {
		pathExpr, value, err := data.SplitSetString(ss)
		if err != nil {
			return fmt.Errorf("error parsing --set value '%s': %w", ss, err)
		}
		parsed, err := jsonpath.Parse(pathExpr)
		if err != nil {
			return fmt.Errorf("error parsing --set value '%s': %w", ss, err)
		}
		pq := parsed.Query().Singular()
		if pq == nil {
			return fmt.Errorf("error parsing --set value '%s': resulting path is not unique singular path", ss)
		}

		err = data.SetValueInData(mergedData, parsed.Query().Segments(), value, ss)
		if err != nil {
			return err
		}

		app.Logger.Debug().Msg(fmt.Sprintf("set %s to %v\n", parsed, value))
	}

	commonTemplates := data.LoadSharedTemplates(app.Settings.CommonTemplateFiles, app.Logger)
	templates, err := yutcTemplate.LoadTemplates(templateFiles, commonTemplates, app.Settings.Strict, app.Logger)
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
					return err
				}
				// no other work needed since it's just a directory, moving on
				continue
			}
			if inputIsRecursive {
				outputPath = files.NormalizeFilepath(filepath.Join(app.Settings.Output, relativePath))
			} else {
				outputPath = files.NormalizeFilepath(app.Settings.Output)
			}
			if tmpl == nil {
				return fmt.Errorf("template is nil, this should never happen but haven't fully tested this yet to be sure")
			}
		}
		outData := new(bytes.Buffer)
		err = tmpl.Execute(outData, mergedData)
		if err != nil {
			return err
		}
		if app.Settings.Output == "-" {
			app.Logger.Debug().Msg("Writing to stdout")
			_, err = os.Stdout.Write(outData.Bytes())
			if err != nil {
				return err
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
					return err
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
					return err
				}
			} else {
				app.Logger.Error().Msg("file exists and overwrite is not set: " + outputPath)
			}
		}
	}
	return err
}

// ResolveFileOutput resolves the output path for a file relative to a base directory.
// If nestedBase is empty, returns inputPath unchanged.
// If inputPath equals nestedBase, returns ".".
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

// LogSettings logs the current application settings as YAML at TRACE level.
func (app *App) LogSettings() {
	app.Logger.Trace().Msg("Settings:")
	yamlSettings, err := yaml.Marshal(app.Settings)
	if err != nil {
		app.Logger.Panic().Msg(err.Error()) // this should never happen unless we seriously goofed up
	}
	for _, line := range bytes.Split(yamlSettings, []byte("\n")) {
		app.Logger.Trace().Msg("  " + string(line))
	}
}

// TemplateFilenames executes a template on a filename and returns the result.
// This allows dynamic filename generation based on template data.
func TemplateFilenames(outputPath string, commonTemplates []*bytes.Buffer, data map[string]any, strict bool, logger *zerolog.Logger) string {
	filenameTemplate, err := yutcTemplate.BuildTemplate(outputPath, commonTemplates, "filename", strict)
	if err != nil {
		logger.Panic().Msg(err.Error())
		return ""
	}
	if filenameTemplate == nil {
		err = fmt.Errorf("error building filename template for %s", outputPath)
		logger.Panic().Msg(err.Error())
		return ""
	}
	templatedPath := new(bytes.Buffer)
	err = filenameTemplate.Execute(templatedPath, data)
	if err != nil {
		logger.Panic().Msg(err.Error())
		return ""
	}
	return templatedPath.String()
}
