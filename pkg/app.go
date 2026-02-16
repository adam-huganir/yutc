// Package yutc provides the core application logic for the yutc template processor.
package yutc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/adam-huganir/yutc/pkg/config"
	"github.com/adam-huganir/yutc/pkg/data"
	"github.com/adam-huganir/yutc/pkg/loader"
	yutcTemplate "github.com/adam-huganir/yutc/pkg/templates"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/goccy/go-yaml"
	"github.com/rs/zerolog"
)

// App holds the application state and dependencies for template execution.
type App struct {
	Settings *types.Arguments
	RunData  *RunData
	Logger   *zerolog.Logger
	TempDir  string
}

// NewApp creates a new App instance with the provided settings, run data, and logger.
// It also generates a unique temporary directory name for the application run.
func NewApp(settings *types.Arguments, runData *RunData, logger *zerolog.Logger) *App {
	tempDir, err := loader.GenerateTempDirName("yutc-*")
	if err != nil {
		logger.Error().Err(err).Msg("failed to generate temp directory name")
	}
	return &App{
		Settings: settings,
		RunData:  runData,
		Logger:   logger,
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

	// grab the name of a temp directory to use for processing, but it is not guaranteed to exist yet
	tempDir := app.TempDir
	defer func() {
		if exists, err := loader.Exists(tempDir); exists {
			if err != nil {
				app.Logger.Error().Err(err).Msg("failed to check if temp directory exists")
			}
			_ = os.RemoveAll(tempDir)
		}
	}()

	app.RunData.TemplateFiles, err = yutcTemplate.ResolveTemplatePaths(app.Settings.TemplatePaths, false, app.Logger)
	if err != nil {
		return err
	}
	app.RunData.CommonTemplateFiles, err = yutcTemplate.ResolveTemplatePaths(app.Settings.CommonTemplateFiles, true, app.Logger)
	if err != nil {
		return err
	}
	app.RunData.DataFiles, err = data.ResolveDataPaths(app.Settings.DataFiles, app.Logger)
	if err != nil {
		return err
	}

	// Filter out common template data from the main template list to avoid duplicate loading
	// we make assumption that the intention of anything specified as a common template explicitly
	// will not intend for it to be loaded again or copied even if it was included in the main template paths
	app.RunData.TemplateFiles = filterCommonTemplateInputs(app.RunData.TemplateFiles, app.RunData.CommonTemplateFiles)

	err = config.ValidateArguments(app.Settings, &config.ParsedInputs{
		DataFiles:           app.RunData.DataFiles,
		TemplateFiles:       app.RunData.TemplateFiles,
		CommonTemplateFiles: app.RunData.CommonTemplateFiles,
	}, app.Logger)
	if err != nil {
		return err
	}

	app.RunData.MergedData, err = data.MergeDataFiles(app.RunData.DataFiles, app.Settings.SetData, app.Settings.Helm, app.Logger)
	if err != nil {
		return err
	}

	templateSet, err := yutcTemplate.LoadTemplateSet(
		app.RunData.TemplateFiles,
		app.RunData.CommonTemplateFiles,
		app.RunData.MergedData,
		app.Settings.Strict,
		app.Settings.IncludeFilenames,
		app.Settings.DropExtension,
		app.Logger,
	)
	if err != nil {
		return err
	}

	// Execute each template from the shared template object
	var skip []string

	for _, templateFile := range templateSet.TemplateFiles {
		templatePath := templateFile.Name // The template name (file path)
		if templateFile.Template.NewName != "" {
			templatePath = templateFile.Template.NewName
		}

		// Compute relative path from the root container if it exists
		relativePath, err := templateFile.RelativeNewPath()
		if err != nil {
			return err
		}

		var outputPath string
		if app.Settings.Output != "-" {
			outputIsDir, err := loader.IsDir(app.Settings.Output)
			if err != nil {
				// If output doesn't exist, treat as directory if we have multiple files
				if len(templateSet.TemplateFiles) > 1 {
					outputIsDir = true
				}
			}

			if outputIsDir {
				outputPath = loader.NormalizeFilepath(filepath.Join(app.Settings.Output, relativePath))
			} else {
				outputPath = loader.NormalizeFilepath(app.Settings.Output)
			}
		}
		outData := new(bytes.Buffer)
		// execute the specific named template from the shared template object
		if slices.Contains(skip, templatePath) {
			return fmt.Errorf(
				"template %s was marked to be skipped for processing, but is being preocessed, report a bug ticket please",
				templatePath,
			)
		}
		err = templateSet.Template.ExecuteTemplate(outData, templatePath, app.RunData.MergedData)
		if err != nil {
			return &types.TemplateError{
				TemplatePath: templatePath,
				Err:          err,
			}
		}
		outBytes := outData.Bytes()
		switch app.Settings.Output {
		case "-":
			app.Logger.Debug().Msg("Writing to stdout")
			_, err = os.Stdout.Write(outBytes)
			if err != nil {
				return err
			}
		default:
			if app.Settings.IgnoreEmpty && strings.TrimSpace(string(outBytes)) == "" {
				app.Logger.Debug().Msgf("Skipping empty output for template: %s", templatePath)
				continue
			}
			_ = filepath.Dir(outputPath)
			outputBasename := filepath.Base(outputPath)

			isDir, err := loader.IsDir(outputPath)
			if err == nil && isDir && len(templateSet.TemplateFiles) == 1 {
				// behavior for single template file and output is a directory
				// matches normal behavior expected by commands like cp, mv etc.
				outputPath = filepath.Join(app.Settings.Output, outputBasename)
				_, err = loader.IsDir(outputPath)
				if err != nil {
					return err
				}
			}

			isDir, err = loader.IsDir(outputPath)
			// the error here is going to be that the file doesn't exist
			if err != nil || (!isDir && app.Settings.Overwrite) {
				if app.Settings.Overwrite {
					app.Logger.Debug().Msg("Overwrite enabled, writing to file(s): " + app.Settings.Output)
				}
				err = os.MkdirAll(filepath.Dir(outputPath), 0o755)
				if err != nil {
					return err
				}
				err = os.WriteFile(outputPath, outBytes, 0o644)
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

// filterCommonTemplateInputs removes entries from templateFiles that are present in commonFiles.
// This prevents duplicate loading of templates that are already loaded as common/shared templates.
func filterCommonTemplateInputs(templateFiles, commonFiles []*yutcTemplate.Input) []*yutcTemplate.Input {
	// Create a map for de-duplication
	commonFilesMap := make(map[string]bool, len(commonFiles))
	for _, cf := range commonFiles {
		normalized := loader.NormalizeFilepath(cf.Name)
		commonFilesMap[normalized] = true
	}

	// Filter out common data from template data
	filtered := make([]*yutcTemplate.Input, 0, len(templateFiles))
	for _, tf := range templateFiles {
		normalized := loader.NormalizeFilepath(tf.Name)
		if !commonFilesMap[normalized] {
			filtered = append(filtered, tf)
		}
	}
	return filtered
}

// RunData holds runtime data for template execution including data files and template paths.
type RunData struct {
	DataFiles           []*data.Input
	CommonTemplateFiles []*yutcTemplate.Input
	TemplateFiles       []*yutcTemplate.Input
	MergedData          map[string]any
}
