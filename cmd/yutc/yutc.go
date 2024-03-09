package main

import (
	"bytes"
	"errors"
	"flag"
	"github.com/adam-huganir/yutc/internal"
	"os"
	"path/filepath"
	"strconv"
)

var logger = internal.GetLogHandler()

func main() {
	var err error
	// Define flags
	var stdin, stdinFirst, overwrite, noStdin, noStdinFirst, noOverwrite, version bool
	var dataFiles internal.RepeatedStringFlag
	var output string

	flag.Usage = internal.OverComplicatedHelp
	flag.BoolVar(&version, "version", false, internal.HelpMessages["version"])
	flag.BoolVar(&stdin, "stdin", false, internal.HelpMessages["stdin"])
	flag.BoolVar(&noStdin, "no-stdin", true, "Do not "+internal.HelpMessages["stdin"])
	flag.BoolVar(&stdinFirst, "stdin-first", false, internal.HelpMessages["stdin-first"])
	flag.BoolVar(&noStdinFirst, "no-stdin-first", true, "Do not "+internal.HelpMessages["stdin-first"])
	flag.Var(&dataFiles, "data", internal.HelpMessages["data"])
	flag.StringVar(&output, "output", "", internal.HelpMessages["output"])
	flag.BoolVar(&overwrite, "overwrite", false, internal.HelpMessages["overwrite"])
	flag.BoolVar(&noOverwrite, "no-overwrite", true, "Do not "+internal.HelpMessages["overwrite"])
	flag.Parse()
	templateFiles := flag.Args()

	if version {
		internal.PrintVersion()
		os.Exit(0)
	}

	validateArguments(
		stdin,
		stdinFirst,
		overwrite,
		noStdin,
		noStdinFirst,
		noOverwrite,
		dataFiles,
		templateFiles,
		output,
	)

	settings := internal.CLIOptions{
		Stdin:         stdin,
		NoStdin:       noStdin,
		DataFiles:     dataFiles,
		TemplateFiles: templateFiles,
		Output:        output,
		Overwrite:     overwrite,
		NoOverwrite:   noOverwrite,
		StdinFirst:    stdinFirst,
		NoStdinFirst:  noStdinFirst,
	}

	// TODO: replace top level panics with proper error handling
	var inData *bytes.Buffer
	if stdin {
		inData, err = internal.GetDataFromFile(os.Stdin)
	}
	data, err := internal.MergeData(settings, inData)
	if err != nil {
		panic(err)
	}
	templates, err := internal.LoadTemplates(settings)
	for templateIndex, tmpl := range templates {
		var outData *bytes.Buffer
		outData = new(bytes.Buffer)
		err = tmpl.Execute(outData, data)
		if err != nil {
			panic(err)
		}
		basename := filepath.Base(settings.TemplateFiles[templateIndex])
		if settings.Output != "" {
			logger.Debug("Writing to file(s) at: " + settings.Output)
			var outputPath string
			if len(templates) > 1 {
				outputPath = filepath.Join(settings.Output, basename)
			} else {
				outputPath = settings.Output
			}
			if err != nil {
				panic(err)
			}
			_, err := os.Stat(outputPath)
			if err != nil {
				if os.IsNotExist(err) {
					err = os.WriteFile(outputPath, outData.Bytes(), 0644)
					if err != nil {
						panic(err)
					}
				} else {
					panic(err)
				}
			} else {
				if settings.Overwrite {
					err = os.WriteFile(outputPath, outData.Bytes(), 0644)
					if err != nil {
						panic(err)
					}
				} else {
					logger.Error("file exists and overwrite is not set: " + outputPath)
				}
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

func validateArguments(
	stdin,
	stdinFirst,
	overwrite,
	noStdin,
	noStdinFirst,
	noOverwrite bool,
	dataFiles,
	templateFiles []string,
	output string,
) {
	var err error
	var errs []error
	var code, v int64

	if len(templateFiles) == 0 {
		err = errors.New("must provide at least one template file")
		v, _ = strconv.ParseInt("1", 2, 64)
		code += v
		errs = append(errs, err)
	}

	if stdin && len(dataFiles) != 0 {
		err = errors.New("cannot use `stdin` with data files")
		v, _ = strconv.ParseInt("10", 2, 64)
		code += v
		errs = append(errs, err)
	}

	outputFiles := output != ""
	if !outputFiles && len(templateFiles) > 1 {
		err = errors.New("cannot use `stdout` with multiple template files")
		v, _ = strconv.ParseInt("100", 2, 64)
		code += v
		errs = append(errs, err)
	}

	if !outputFiles {
		_, err = os.Stat(output)
		if err != nil {
			if os.IsNotExist(err) && len(templateFiles) > 1 {
				err = errors.New("folder " + output + " does not exist to generate multiple templates")
				v, _ = strconv.ParseInt("1000", 2, 64)
				code += v
				errs = append(errs, err)
			}
		}
	}

	if (stdin && noStdin) ||
		(stdinFirst && noStdinFirst) ||
		(overwrite && noOverwrite) {
		err = errors.New("cannot use both `xxx` and `no-xxx for any flags`")
		v, _ = strconv.ParseInt("10000", 2, 64)
		code += v
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		for _, err := range errs {
			logger.Error(err.Error())
		}
		os.Exit(int(code))
	}
}
