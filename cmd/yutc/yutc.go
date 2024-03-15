package main

import (
	"bytes"
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
)

var YutcLog = &internal.YutcLog

func main() {
	var err error
	// Define flags
	var overwrite, version bool
	var dataFiles, commonTemplateFiles []string
	var commonTemplates []*bytes.Buffer
	var output string

	internal.InitLogger()
	YutcLog.Trace().Msg("Starting yutc")

	pflag.StringArrayVarP(
		&dataFiles,
		"data",
		"d",
		nil,
		"Data file to parse and merge. Can be a file or a URL. "+
			"Can be specified multiple times and the inputs will be merged.",
	)
	pflag.StringArrayVarP(
		&commonTemplateFiles,
		"common-templates",
		"c",
		nil,
		"Templates to be shared across all arguments in template list. Can be a file or a URL. Can be specified multiple times.",
	)
	pflag.StringVarP(&output, "output", "o", "-", "Output file/directory, defaults to stdout")
	pflag.BoolVarP(&overwrite, "overwrite", "w", false, "Overwrite existing files")
	pflag.BoolVar(&version, "version", false, "Print the version and exit")
	pflag.Parse()
	templateFiles := pflag.Args()
	if version {
		internal.PrintVersion()
		os.Exit(0)
	}

	YutcLog.Trace().Msg("Settings:")
	pflag.VisitAll(func(flag *pflag.Flag) {
		YutcLog.Trace().Msgf("\t%s: %s", flag.Name, flag.Value.String())
	})

	valCode := internal.ValidateArguments(
		dataFiles, commonTemplateFiles, templateFiles, output, overwrite,
	)
	if valCode > 0 {
		YutcLog.Error().Msg("Invalid arguments")
		os.Exit(int(valCode))
	}

	data, err := internal.MergeData(dataFiles)
	if err != nil {
		panic(err)
	}

	commonTemplates = internal.LoadSharedTemplates(commonTemplateFiles)

	templates, err := internal.LoadTemplates(templateFiles, commonTemplates)
	for templateIndex, tmpl := range templates {
		var outData *bytes.Buffer
		outData = new(bytes.Buffer)
		err = tmpl.Execute(outData, data)
		if err != nil {
			panic(err)
		}
		basename := filepath.Base(templateFiles[templateIndex])
		// stdin not handled here, gotta do that
		if output != "-" {
			var outputPath string
			if len(templates) > 1 {
				outputPath = filepath.Join(output, basename)
			} else {
				outputPath = output
			}
			if err != nil {
				panic(err)
			}
			isDir, err := checkIfDir(outputPath)
			if err == nil && *isDir && len(templates) == 1 {
				// behavior for single template file and output is a directory
				// matches normal behavior expected by commands like cp, mv etc.
				outputPath = filepath.Join(output, basename)
			}
			// check again in case the output path was changed and the file still exists,
			// we can probably make this into just one case statement but it's late and i am tired
			isDir, err = checkIfDir(outputPath)
			// error here is going to be that the file doesnt exist
			if err != nil || (!*isDir && overwrite) {
				YutcLog.Fatal().Msg("Writing to file(s) to: " + output)
				err = os.WriteFile(outputPath, outData.Bytes(), 0o644)
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
