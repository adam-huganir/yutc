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
)

type App struct {
	ctx context.Context
}

func NewApp(ctx context.Context) *App {
	tempDir, _ := files.GenerateTempDirName("yutc-*")
	ctx = context.WithValue(ctx, config.TempDirKey, tempDir)
	return &App{ctx: ctx}
}

func (app *App) GetContext() context.Context {
	return app.ctx
}

func (app *App) GetSettings() *types.Arguments {
	return config.GetSettings(app.ctx)
}

func (app *App) GetData() *types.RunData {
	return config.GetRunData(app.ctx)
}

func (app *App) GetLogger() zerolog.Logger {
	return config.GetLogger(app.ctx)
}

func (app *App) GetCommand() *cobra.Command {
	return config.GetCommand(app.ctx)
}

func (app *App) GetTempDir() string {
	return config.GetTempDir(app.ctx)
}

func (app *App) Run(ctx context.Context, args []string) (err error) {
	settings := app.GetSettings()
	logger := app.GetLogger()
	runData := app.GetData()

	settings.TemplatePaths = args
	if logger.GetLevel() < zerolog.DebugLevel {
		app.LogSettings()
	}

	if settings.Version {
		PrintVersion()
		return nil
	}

	if len(settings.TemplatePaths) == 0 {
		logger.Fatal().Msg("No template files specified")
	}

	// grab the name of a temp directory to use for processing
	// but it is not guaranteed to exist yet
	tempDir := app.GetTempDir()
	defer func() {
		if exists, _ := files.Exists(tempDir); exists {
			_ = os.RemoveAll(tempDir)
		}
	}()

	templateFiles, err := data.LoadTemplates(ctx)
	if err != nil {
		return err
	}

	err = app.GetData().ParseDataFiles(settings.DataFiles)
	if err != nil {
		return err
	}
	dataFiles, err := data.LoadDataFiles(ctx)
	if err != nil {
		return err
	}

	commonFiles, _ := files.ResolvePaths(ctx, settings.CommonTemplateFiles)
	err = app.GetData().ParseTemplatePaths(commonFiles)
	if err != nil {
		return err
	}

	exitCode, errs := config.ValidateArguments(ctx)
	if exitCode > 0 {
		var errStrings []string
		for _, err := range errs {
			errStrings = append(errStrings, err.Error())
		}
		return &types.ExitError{Code: exitCode, Err: fmt.Errorf("validation errors: %v", errStrings)}
	}

	mergedData, err := data.MergeData(ctx)
	if err != nil {
		panic(err)
	}
	commonTemplates := data.LoadSharedTemplates(settings.CommonTemplateFiles, logger)
	templates, err := yutcTemplate.LoadTemplates(templateFiles, commonTemplates, settings.Strict, logger)
	if err != nil {
		logger.Panic().Msg(err.Error())
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
		if settings.IncludeFilenames {
			templateOutputPath = TemplateFilenames(templateOriginalPath, commonTemplates, mergedData, settings.Strict, logger)
		}
		if inputIsRecursive {
			relativePath = ResolveFileOutput(templateOutputPath, resolveRoot) // TESTING
		} else if err == nil { // i.e. it's a file
			relativePath = path.Base(templateOutputPath)
		}

		var outputPath string
		if settings.Output != "-" {
			outputIsDir, _ := files.IsDir(templateOriginalPath)
			if outputIsDir {
				outputPath = files.NormalizeFilepath(filepath.Join(settings.Output, relativePath))
				err = os.MkdirAll(outputPath, 0755)
				if err != nil {
					panic(err)
				}
				// no other work needed since it's just a directory, moving on
				continue
			} else {
				if inputIsRecursive {
					outputPath = files.NormalizeFilepath(filepath.Join(settings.Output, relativePath))
				} else {
					outputPath = files.NormalizeFilepath(filepath.Join(settings.Output))
				}
			}
			if tmpl == nil {
				panic("template is nil, this should never happen but haven't fully tested this yet to be sure")
			}
		}
		outData := new(bytes.Buffer)
		err = tmpl.Execute(outData, mergedData)
		if err != nil {
			logger.Panic().Msg(err.Error())
		}
		if settings.Output == "-" {
			logger.Debug().Msg("Writing to stdout")
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
				outputPath = filepath.Join(settings.Output, outputBasename)
				_, err = files.IsDir(outputPath)
				if err != nil {
					logger.Fatal().Msg(err.Error())
				}
			}

			// check again in case the output path was changed and the file still exists,
			// we can probably make this into just one case statement but it's late and i am tired
			if settings.IncludeFilenames {
				outputPath = TemplateFilenames(outputPath, commonTemplates, mergedData, settings.Strict, logger)
			}
			isDir, err = files.IsDir(outputPath)
			// the error here is going to be that the file doesn't exist
			if err != nil || (!isDir && settings.Overwrite) {
				if settings.Overwrite {
					logger.Debug().Msg("Overwrite enabled, writing to file(s): " + settings.Output)
				}
				err = os.WriteFile(outputPath, outData.Bytes(), 0644)
				if err != nil {
					panic(err)
				}
			} else {
				logger.Error().Msg("file exists and overwrite is not set: " + outputPath)
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
	logger.Trace().Msg("Settings:")
	yamlSettings, err := yaml.Marshal(settings)
	if err != nil {
		panic(err) // this should never happen unless we seriously goofed up
	}
	for _, line := range bytes.Split(yamlSettings, []byte("\n")) {
		logger.Trace().Msg("  " + string(line))
	}
}

func TemplateFilenames(outputPath string, commonTemplates []*bytes.Buffer, data map[string]any, strict bool, logger zerolog.Logger) string {
	filenameTemplate, err := yutcTemplate.BuildTemplate(outputPath, commonTemplates, "filename", strict)
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
