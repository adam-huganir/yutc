package main

import (
	"bytes"
	"context"
	"github.com/adam-huganir/yutc/internal"
	"github.com/adam-huganir/yutc/pkg/LoggingUtils"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
)

var logger = LoggingUtils.GetLogHandler()

func main() {
	var err error
	// Define flags
	var overwrite, version bool
	var dataFiles, commonTemplateFiles []string
	var commonTemplates []*bytes.Buffer
	var output string

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

	if LoggingUtils.GetLogLevel() == LoggingUtils.LogLevelTrace {
		logger.Trace("Settings:")
		pflag.VisitAll(func(flag *pflag.Flag) {
			logger.Trace(flag.Name + ": " + flag.Value.String())
		})
	}

	valCode := internal.ValidateArguments(
		dataFiles, commonTemplateFiles, templateFiles, output, overwrite,
	)
	if valCode > 0 {
		logger.Error("Invalid arguments")
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
				logger.Log(context.Background(), LoggingUtils.LogLevelFatal, "Writing to file(s) to: "+output)
				err = os.WriteFile(outputPath, outData.Bytes(), 0o644)
				if err != nil {
					panic(err)
				}
			} else {
				logger.Error("file exists and overwrite is not set: " + outputPath)
			}
		} else {
			logger.Debug("Writing to stdout")
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
		logger.Error(err.Error())
	}
	if stat.IsDir() {
		b = true
	} else {
		b = false
	}
	return &b, nil
}
