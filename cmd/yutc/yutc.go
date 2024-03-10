package main

import (
	"bytes"
	"context"
	"flag"
	"github.com/adam-huganir/yutc/internal"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var logger = internal.GetLogHandler()

func main() {
	var err error
	// Define flags
	var stdin, stdinFirst, overwrite, version bool
	var dataFiles, sharedTemplates internal.RepeatedStringFlag
	var sharedTemplateBuffers []*bytes.Buffer
	var output string

	flag.Usage = internal.OverComplicatedHelp
	flag.BoolVar(&version, "version", false, internal.HelpMessages["version"])
	flag.BoolVar(&stdin, "stdin", false, internal.HelpMessages["stdin"])
	flag.BoolVar(&stdinFirst, "stdin-first", false, internal.HelpMessages["stdin-first"])
	flag.Var(&dataFiles, "data", internal.HelpMessages["data"])
	flag.Var(&sharedTemplates, "shared", internal.HelpMessages["shared"])
	flag.StringVar(&output, "output", "", internal.HelpMessages["output"])
	flag.BoolVar(&overwrite, "overwrite", false, internal.HelpMessages["overwrite"])
	flag.Parse()
	templateFiles := flag.Args()

	if version {
		internal.PrintVersion()
		os.Exit(0)
	}

	settings := internal.CLIOptions{
		Stdin:           stdin,
		DataFiles:       dataFiles,
		TemplateFiles:   templateFiles,
		Output:          output,
		Overwrite:       overwrite,
		SharedTemplates: sharedTemplates,
		StdinFirst:      stdinFirst,
	}
	if internal.GetLogLevel() == internal.LogLevelTrace {
		b, err := yaml.Marshal(settings)
		if err != nil {
			panic(err)
		}
		logger.Debug("Settings:")
		println(string(b))
	}

	internal.ValidateArguments(
		stdin,
		stdinFirst,
		overwrite,
		sharedTemplates,
		dataFiles,
		templateFiles,
		output,
	)

	// TODO: replace top level panics with proper error handling
	var inData *bytes.Buffer
	if stdin {
		inData, err = internal.GetDataFromFile(os.Stdin)
	}
	for _, sharedTemplate := range sharedTemplates {
		sharedTemplateBuffer, err := internal.GetDataFromPath(sharedTemplate)
		if err != nil {
			panic(err)
		}
		sharedTemplateBuffers = append(sharedTemplateBuffers, sharedTemplateBuffer)

	}
	data, err := internal.MergeData(settings, inData)
	if err != nil {
		panic(err)
	}
	templates, err := internal.LoadTemplates(settings, sharedTemplateBuffers)
	for templateIndex, tmpl := range templates {
		var outData *bytes.Buffer
		outData = new(bytes.Buffer)
		err = tmpl.Execute(outData, data)
		if err != nil {
			panic(err)
		}
		basename := filepath.Base(settings.TemplateFiles[templateIndex])
		if settings.Output != "" {
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
				logger.Log(context.Background(), internal.LogLevelFatal, "Writing to file(s) to: "+settings.Output)
				err = os.WriteFile(outputPath, outData.Bytes(), 0644)
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
