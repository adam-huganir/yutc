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
	yutcTemplate "github.com/adam-huganir/yutc/pkg/templates"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/goccy/go-yaml"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/theory/jsonpath"
)

// App holds the application state and dependencies for template execution.
type App struct {
	Settings *types.Arguments
	RunData  *RunData
	Logger   *zerolog.Logger
	Command  *cobra.Command
	TempDir  string
}

// NewApp creates a new App instance with the provided settings, run data, logger, and command.
// It also generates a unique temporary directory name for the application run.
func NewApp(settings *types.Arguments, runData *RunData, logger *zerolog.Logger, cmd *cobra.Command) *App {
	tempDir, err := data.GenerateTempDirName("yutc-*")
	if err != nil {
		logger.Error().Err(err).Msg("failed to generate temp directory name")
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
// It loads data data, parses templates, and generates output based on the configured settings.
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
		app.Logger.Fatal().Msg("No template data specified")
	}

	// grab the name of a temp directory to use for processing, but it is not guaranteed to exist yet
	tempDir := app.TempDir
	defer func() {
		if exists, err := data.Exists(tempDir); exists {
			if err != nil {
				app.Logger.Error().Err(err).Msg("failed to check if temp directory exists")
			}
			_ = os.RemoveAll(tempDir)
		}
	}()

	app.RunData.TemplatePaths, err = data.ResolvePaths(app.Settings.TemplatePaths, data.FileKindTemplate, tempDir, app.Logger)
	if err != nil {
		return err
	}
	app.RunData.CommonTemplateFiles, err = data.ResolvePaths(app.Settings.CommonTemplateFiles, data.FileKindCommonTemplate, tempDir, app.Logger)
	if err != nil {
		return err
	}
	app.RunData.DataFiles, err = data.ResolvePaths(app.Settings.DataFiles, data.FileKindData, tempDir, app.Logger)
	if err != nil {
		return err
	}

	// Filter out common template data from the main template list to avoid duplicate loading
	// we make assumption that the intention of anything specified as a common template explicitly
	// will not intend for it to be loaded again or copied even if it was included in the main template paths
	app.RunData.TemplatePaths = filterCommonFileArgs(app.RunData.TemplatePaths, app.RunData.CommonTemplateFiles)

	err = config.ValidateArguments(app.Settings, app.Logger)
	if err != nil {
		return err
	}

	mergedData, err := data.MergeDataFiles(app.RunData.DataFiles, app.Settings.Helm, app.Logger)
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
		if pq := parsed.Query().Singular(); pq == nil {
			return fmt.Errorf("error parsing --set value '%s': resulting path is not unique singular path", ss)
		}
		var mergedDataAny any
		mergedDataAny = mergedData
		err = data.SetValueInData(&mergedDataAny, parsed.Query().Segments(), value, ss)
		if err != nil {
			return err
		}

		app.Logger.Debug().Msg(fmt.Sprintf("set %s to %v\n", parsed, value))
	}

	commonTemplates := make([]*bytes.Buffer, len(app.RunData.CommonTemplateFiles))
	for i, f := range app.RunData.CommonTemplateFiles {
		commonTemplates[i] = bytes.NewBuffer(f.Content.Data)
	}

	templateSet, err := yutcTemplate.LoadTemplateSet(app.RunData.TemplatePaths, commonTemplates, app.Settings.Strict, app.Logger)
	if err != nil {
		return err
	}

	// we rely on validation to make sure we aren't getting multiple recursables
	firstTemplate := app.RunData.TemplatePaths[0]
	inputIsContainer, err := firstTemplate.IsContainer()

	// TODO: figure out what this code is for, i suspect with recent changes it needs to be redone
	resolveRoot := ""
	if err == nil && inputIsContainer {
		resolveRoot = firstTemplate.Path
	}

	// Execute each template from the shared template object
	var skip []string

	for _, templateItem := range templateSet.TemplateItems {
		templateOriginalPath := templateItem.Path // The template name (file path)

		// if we have a directory as our template source we want to keep track of relative paths
		// execute filenames as templates if requested
		var relativePath string
		if app.Settings.IncludeFilenames {
			filenameTemplate, err := yutcTemplate.InitTemplate(commonTemplates, app.Settings.Strict)
			if err != nil {
				return fmt.Errorf("error initializing filename template: %w", err)
			}
			newName, err := yutcTemplate.TemplateFilenames(filenameTemplate, templateOriginalPath, commonTemplates, mergedData, app.Logger)
			if err != nil {
				return fmt.Errorf("error parsing template filenames: %w", err)
			}
			if newName == "" {
				return fmt.Errorf("templated filename for %s resulted in empty string, cannot continue", templateOriginalPath)
			}
			if newName != templateItem.Path {
				// reparse the template now that the name has been changed by templating
				templateItem.Path = newName
				_, err = templateSet.Template.New(templateItem.Path).Parse(string(templateItem.Content.Data))
				if err != nil {
					return &types.TemplateError{
						TemplatePath: templateOriginalPath,
						Err:          fmt.Errorf("error parsing template after applying filename templating: %w", err),
					}
				}

				skip = append(skip, templateOriginalPath) // just to be extra sure that future updates won't re-process this
			}
		}
		if inputIsContainer {
			relativePath = ResolveFileOutput(templateItem.Path, resolveRoot)
		} else if err == nil { // i.e. it's a file
			relativePath = path.Base(templateItem.Path)
		}

		var outputPath string
		if app.Settings.Output != "-" {
			outputIsDir, err := data.IsDir(templateOriginalPath)
			if err != nil {
				return err
			}
			if outputIsDir {
				outputPath = data.NormalizeFilepath(filepath.Join(app.Settings.Output, relativePath))
				err = os.MkdirAll(outputPath, 0o755)
				if err != nil {
					return err
				}
				// no other work needed since it's just a directory, moving on
				continue
			}
			if inputIsContainer {
				outputPath = data.NormalizeFilepath(filepath.Join(app.Settings.Output, relativePath))
			} else {
				outputPath = data.NormalizeFilepath(app.Settings.Output)
			}
		}
		outData := new(bytes.Buffer)
		// execute the specific named template from the shared template object
		if slices.Contains(skip, templateItem.Path) {
			return fmt.Errorf(
				"template %s was marked to be skipped for processing, but is being preocessed, report a bug ticket please",
				templateItem.Path,
			)
		}
		err = templateSet.Template.ExecuteTemplate(outData, templateItem.Path, mergedData)
		if err != nil {
			return &types.TemplateError{
				TemplatePath: templateOriginalPath,
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
				app.Logger.Debug().Msgf("Skipping empty output for template: %s", templateOriginalPath)
				continue
			}
			_ = filepath.Dir(outputPath)
			outputBasename := filepath.Base(outputPath)

			isDir, err := data.IsDir(outputPath)
			if err == nil && isDir && len(templateSet.TemplateItems) == 1 {
				// behavior for single template file and output is a directory
				// matches normal behavior expected by commands like cp, mv etc.
				outputPath = filepath.Join(app.Settings.Output, outputBasename)
				_, err = data.IsDir(outputPath)
				if err != nil {
					return err
				}
			}

			// check again in case the output path was changed and the file still exists,
			// we can probably make this into just one case statement, but it's late and i am tired
			if app.Settings.IncludeFilenames {
				filenameTemplate, err := yutcTemplate.InitTemplate(commonTemplates, app.Settings.Strict)
				if err != nil {
					return fmt.Errorf("error initializing filename template: %w", err)
				}
				outputPath, err = yutcTemplate.TemplateFilenames(filenameTemplate, outputPath, commonTemplates, mergedData, app.Logger)
				if err != nil {
					return err
				}
			}
			isDir, err = data.IsDir(outputPath)
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

// filterCommonFileArgs removes data from templateFiles that are present in commonFiles.
// This prevents duplicate loading of templates that are already loaded as common/shared templates.
func filterCommonFileArgs(templateFiles, commonFiles []*data.FileArg) []*data.FileArg {
	// Create a map for de-duplication
	commonFilesMap := make(map[string]bool, len(commonFiles))
	for _, cf := range commonFiles {
		normalized := data.NormalizeFilepath(cf.Path)
		commonFilesMap[normalized] = true
	}

	// Filter out common data from template data
	filtered := make([]*data.FileArg, 0, len(templateFiles))
	for _, tf := range templateFiles {
		normalized := data.NormalizeFilepath(tf.Path)
		if !commonFilesMap[normalized] {
			filtered = append(filtered, tf)
		}
	}
	return filtered
}

// RunData holds runtime data for template execution including data data and template paths.
type RunData struct {
	DataFiles           []*data.FileArg
	CommonTemplateFiles []*data.FileArg
	TemplatePaths       []*data.FileArg
}
