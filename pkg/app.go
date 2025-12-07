// Package yutc provides the core application logic for the yutc template processor.
package yutc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/adam-huganir/yutc/pkg/config"
	"github.com/adam-huganir/yutc/pkg/data"
	"github.com/adam-huganir/yutc/pkg/files"
	yutcTemplate "github.com/adam-huganir/yutc/pkg/templates"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
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
	tempDir, err := files.GenerateTempDirName("yutc-*")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate temp directory name")
	}
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
		if exists, err := files.Exists(tempDir); exists {
			if err != nil {
				app.Logger.Error().Err(err).Msg("Failed to check if temp directory exists")
			}
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

	commonFiles, err := files.ResolvePaths(app.Settings.CommonTemplateFiles, tempDir, app.Logger)
	if err != nil {
		return err
	}
	err = data.ParseTemplatePaths(app.RunData, commonFiles)
	if err != nil {
		return err
	}

	// Filter out common template files from the main template list to avoid duplicate loading
	// we make assumption that the intention of anything specified as a common template explicitly
	// will not intend for it to be loaded again or copied even if it was included in the main template paths
	templateFiles = filterOutCommonFiles(templateFiles, commonFiles)

	err = config.ValidateArguments(app.Settings, app.Logger)
	if err != nil {
		return err
	}

	mergedData, err := data.MergeData(dataFiles, app.Settings.Helm, app.Logger)
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

	commonTemplates, err := data.LoadSharedTemplates(app.Settings.CommonTemplateFiles, app.Logger)
	if err != nil {
		return err
	}
	templateSet, err := yutcTemplate.LoadTemplateSet(templateFiles, commonTemplates, app.Settings.Strict, app.Logger)
	if err != nil {
		return err
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

	// Execute each template from the shared template object
	var skip []string
	for _, templateItem := range templateSet.TemplateItems {
		templateOriginalPath := templateItem.Name // The template name (file path)

		// if we have a directory as our template source we want to keep track of relative paths
		// execute filenames as templates if requested
		var relativePath string
		if app.Settings.IncludeFilenames {
			newName, err := TemplateFilenames(templateOriginalPath, commonTemplates, mergedData, app.Settings.Strict, app.Logger)
			if err != nil {
				return errors.Wrap(err, "error parsing template filenames")
			}
			if newName == "" {
				return fmt.Errorf("templated filename for %s resulted in empty string, cannot continue", templateOriginalPath)
			}
			if newName != templateItem.Name {
				// re-parse the template now that the name has been changed by templating
				templateItem.Name = newName
				_, err = templateSet.Template.New(templateItem.Name).Parse(templateItem.Content.String())
				if err != nil {
					return &types.TemplateError{
						TemplatePath: templateOriginalPath,
						Err:          errors.Wrap(err, "error parsing template after applying filename templating"),
					}
				}

				skip = append(skip, templateOriginalPath) // just to be extra sure that future updates won't re-process this
			}
		}
		if inputIsRecursive {
			relativePath = ResolveFileOutput(templateItem.Name, resolveRoot)
		} else if err == nil { // i.e. it's a file
			relativePath = path.Base(templateItem.Name)
		}

		var outputPath string
		if app.Settings.Output != "-" {
			outputIsDir, err := files.IsDir(templateOriginalPath)
			if err != nil {
				return err
			}
			if outputIsDir {
				outputPath = files.NormalizeFilepath(filepath.Join(app.Settings.Output, relativePath))
				err = os.MkdirAll(outputPath, 0o755)
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
		}
		outData := new(bytes.Buffer)
		// execute the specific named template from the shared template object
		if slices.Contains(skip, templateItem.Name) {
			return fmt.Errorf(
				"template %s was marked to be skipped for processing, but is being preocessed, report a bug ticket please",
				templateItem.Name,
			)
		}
		err = templateSet.Template.ExecuteTemplate(outData, templateItem.Name, mergedData)
		if err != nil {
			return &types.TemplateError{
				TemplatePath: templateOriginalPath,
				Err:          err,
			}
		}
		outBytes := outData.Bytes()
		if app.Settings.Output == "-" {
			app.Logger.Debug().Msg("Writing to stdout")
			_, err = os.Stdout.Write(outBytes)
			if err != nil {
				return err
			}
		} else if app.Settings.IgnoreEmpty && strings.TrimSpace(string(outBytes)) == "" {
			app.Logger.Debug().Msgf("Skipping empty output for template: %s", templateOriginalPath)
			continue
		} else {

			_ = filepath.Dir(outputPath)
			outputBasename := filepath.Base(outputPath)

			isDir, err := files.IsDir(outputPath)
			if err == nil && isDir && len(templateSet.TemplateItems) == 1 {
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
				outputPath, _ = TemplateFilenames(outputPath, commonTemplates, mergedData, app.Settings.Strict, app.Logger)
			}
			isDir, err = files.IsDir(outputPath)
			// the error here is going to be that the file doesn't exist
			if err != nil || (!isDir && app.Settings.Overwrite) {
				if app.Settings.Overwrite {
					app.Logger.Debug().Msg("Overwrite enabled, writing to file(s): " + app.Settings.Output)
				}
				err = os.MkdirAll(filepath.Dir(outputPath), 0o755)
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
func TemplateFilenames(outputPath string, commonTemplates []*bytes.Buffer, mergedData map[string]any, strict bool, logger *zerolog.Logger) (string, error) {
	//filenameTemplate, err := yutcTemplate.BuildTemplate(outputPath, commonTemplates, "filename", strict)
	filenameTemplate, err := yutcTemplate.InitTemplate(commonTemplates, strict)
	if err != nil {
		return "", errors.Wrap(err, "error initializing filename template")
	}
	_, err = filenameTemplate.New(outputPath).Parse(outputPath)
	templatedPath := new(bytes.Buffer)
	err = filenameTemplate.ExecuteTemplate(templatedPath, outputPath, mergedData)
	if err != nil {
		templateErr := &types.TemplateError{
			TemplatePath: outputPath,
			Err:          err,
		}
		return "", templateErr
	}
	return templatedPath.String(), nil
}

// filterOutCommonFiles removes files from templateFiles that are present in commonFiles.
// This prevents duplicate loading of templates that are already loaded as common/shared templates.
func filterOutCommonFiles(templateFiles, commonFiles []string) []string {
	// Create a map for O(1) lookup, using normalized paths
	commonFilesMap := make(map[string]bool, len(commonFiles))
	for _, cf := range commonFiles {
		normalized := files.NormalizeFilepath(cf)
		commonFilesMap[normalized] = true
	}

	// Filter out common files from template files
	filtered := make([]string, 0, len(templateFiles))
	for _, tf := range templateFiles {
		normalized := files.NormalizeFilepath(tf)
		if !commonFilesMap[normalized] {
			filtered = append(filtered, tf)
		}
	}
	return filtered
}
