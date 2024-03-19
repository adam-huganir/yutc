package main

import (
	"bytes"
	"context"
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
)

var YutcLog = &internal.YutcLog
var rootCLI = &cobra.Command{
	Use:     "yutc [ OPTIONS ... ] templates ...",
	Example: "yutc -d data.yaml -c common.tmpl -o output.yaml template.tmpl",
	Args:    cobra.MinimumNArgs(1),
	Run:     run,
}

func initCLI(cmd *cobra.Command) *internal.CLISettings {
	// Define flags
	runSettings := &internal.CLISettings{}

	internal.InitLogger()
	cmd.Flags().SortFlags = false
	cmd.Flags().StringArrayVarP(
		&runSettings.DataFiles,
		"data",
		"d",
		nil,
		"Data file to parse and merge. Can be a file or a URL. "+
			"Can be specified multiple times and the inputs will be merged.",
	)
	cmd.Flags().StringArrayVarP(
		&runSettings.CommonTemplateFiles,
		"common-templates",
		"c",
		nil,
		"Templates to be shared across all arguments in template list. Can be a file or a URL. Can be specified multiple times.",
	)
	cmd.Flags().StringVarP(&runSettings.Output, "output", "o", "-", "Output file/directory, defaults to stdout")
	cmd.Flags().BoolVarP(&runSettings.Overwrite, "overwrite", "w", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&runSettings.Version, "version", false, "Print the version and exit")
	return runSettings
}

func main() {

	internal.InitLogger()
	YutcLog.Trace().Msg("Starting yutc...")

	cmd := rootCLI
	//args := os.Args[1:]
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	settings := initCLI(cmd)
	ctx = context.WithValue(ctx, "settings", settings)

	err := rootCLI.ExecuteContext(ctx)
	if err != nil {
		YutcLog.Error().Msg(err.Error())
		os.Exit(10101)
	}
}

func run(cmd *cobra.Command, args []string) {
	var err error
	settings := cmd.Context().Value("settings").(*internal.CLISettings)
	if err != nil {
		YutcLog.Error().Msg(err.Error())
		//os.Exit(10)
	}
	settings.TemplateFiles = cmd.Flags().Args()
	if settings.Version {
		internal.PrintVersion()
		os.Exit(0)
	}

	YutcLog.Trace().Msg("Settings:")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		YutcLog.Trace().Msgf("\t%s: %s", flag.Name, flag.Value.String())
	})

	valCode := internal.ValidateArguments(settings)
	if valCode > 0 {
		YutcLog.Error().Msg("Invalid arguments")
		os.Exit(int(valCode))
	}

	data, err := internal.MergeData(settings.DataFiles)
	if err != nil {
		panic(err)
	}

	commonTemplates := internal.LoadSharedTemplates(settings.CommonTemplateFiles)

	templates, err := internal.LoadTemplates(settings.TemplateFiles, commonTemplates)
	for templateIndex, tmpl := range templates {
		var outData *bytes.Buffer
		outData = new(bytes.Buffer)
		err = tmpl.Execute(outData, data)
		if err != nil {
			panic(err)
		}
		basename := filepath.Base(settings.TemplateFiles[templateIndex])
		// stdin not handled here, gotta do that
		if settings.Output != "-" {
			var outputPath string
			if len(templates) > 1 {
				outputPath = filepath.Join(settings.Output, basename)
			} else {
				outputPath = settings.Output
			}
			if err != nil {
				panic(err)
			}
			isDir, err := checkIfDir(outputPath)
			if err == nil && *isDir && len(templates) == 1 {
				// behavior for single template file and output is a directory
				// matches normal behavior expected by commands like cp, mv etc.
				outputPath = filepath.Join(settings.Output, basename)
			}
			// check again in case the output path was changed and the file still exists,
			// we can probably make this into just one case statement but it's late and i am tired
			isDir, err = checkIfDir(outputPath)
			// error here is going to be that the file doesnt exist
			if err != nil || (!*isDir && settings.Overwrite) {
				YutcLog.Debug().Msg("Overwrite enabled, writing to file(s): " + settings.Output)
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

func checkIfDir(path string) (*bool, error) {
	var b bool
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		YutcLog.Error().Msg(err.Error())
	}
	if stat.IsDir() {
		b = true
	} else {
		b = false
	}
	return &b, nil
}
