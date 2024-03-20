package main

import (
	"bytes"
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
)

var YutcLog = &internal.YutcLog

var runSettings = &internal.CLISettings{}
var rootCommand = &cobra.Command{
	Use:     "yutc [ OPTIONS ... ] templates ...",
	Example: "yutc -d data.yaml -c common.tmpl -o output.yaml template.tmpl",

	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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

		YutcLog.Trace().Msg("Settings:")
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			YutcLog.Trace().Msgf("\t%s: %s", flag.Name, flag.Value.String())
		})

		valCode := internal.ValidateArguments(runSettings)
		if valCode > 0 {
			YutcLog.Error().Msg("Invalid arguments")
			os.Exit(int(valCode))
		}

		data, err := internal.MergeData(runSettings.DataFiles)
		if err != nil {
			panic(err)
		}

		commonTemplates := internal.LoadSharedTemplates(runSettings.CommonTemplateFiles)

		templates, err := internal.LoadTemplates(runSettings.TemplateFiles, commonTemplates)
		for templateIndex, tmpl := range templates {
			var outData *bytes.Buffer
			outData = new(bytes.Buffer)
			err = tmpl.Execute(outData, data)
			if err != nil {
				panic(err)
			}
			basename := filepath.Base(runSettings.TemplateFiles[templateIndex])
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
	},
}

func init() {

	internal.InitLogger()
	rootCommand.Flags().SortFlags = false
	rootCommand.Flags().StringArrayVarP(
		&runSettings.DataFiles,
		"data",
		"d",
		nil,
		"Data file to parse and merge. Can be a file or a URL. "+
			"Can be specified multiple times and the inputs will be merged.",
	)
	rootCommand.Flags().StringArrayVarP(
		&runSettings.CommonTemplateFiles,
		"common-templates",
		"c",
		nil,
		"Templates to be shared across all arguments in template list. Can be a file or a URL. Can be specified multiple times.",
	)
	rootCommand.Flags().StringVarP(&runSettings.Output, "output", "o", "-", "Output file/directory, defaults to stdout")
	rootCommand.Flags().BoolVarP(&runSettings.Overwrite, "overwrite", "w", false, "Overwrite existing files")
	rootCommand.Flags().BoolVar(&runSettings.Version, "version", false, "Print the version and exit")
}

func main() {

	internal.InitLogger()
	YutcLog.Trace().Msg("Starting yutc...")

	err := rootCommand.Execute()
	if err != nil {
		YutcLog.Error().Msg(err.Error())
		os.Exit(10101)
	}
}
