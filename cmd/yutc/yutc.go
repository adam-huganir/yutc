package main

import (
	"errors"
	"flag"
	"github.com/adam-huganir/yutc/internal"
	"gopkg.in/yaml.v3"
	"os"
	"strconv"
)

var logger = internal.GetLogHandler()

type CLIOptions struct {
	Stdin         bool     `json:"stdin"`
	Stdout        bool     `json:"stdout"`
	DataFiles     []string `json:"data_files"`
	TemplateFiles []string `json:"template_files"`
}

// TODO: replace top level panics with proper error handling

func main() {
	// Define flags
	var stdin, stdout bool
	var dataFiles, templateFiles internal.RepeatedStringFlag

	flag.BoolVar(&stdin, "stdin", false, "Read template from Stdin")
	flag.BoolVar(&stdout, "stdout", true, "Output to Stdout")
	flag.Var(&dataFiles, "data", "Data file to parse and merge")
	flag.Var(&templateFiles, "template", "Template file to parse and merge")
	flag.Parse()
	settings := CLIOptions{
		Stdin:         stdin,
		Stdout:        stdout,
		DataFiles:     dataFiles,
		TemplateFiles: templateFiles,
	}

	validateSettings(settings)

	data := make(map[interface{}]interface{})
	logger.Debug("Loading " + strconv.Itoa(len(settings.DataFiles)) + " data files")
	for _, s := range settings.DataFiles {
		logger.Debug("Data file: " + s)
		path, err := internal.ParseStringFlag(s)
		if err != nil {
			panic(err)
		}
		logger.Debug("Data file path: " + path.String())
		contentBuffer, err := internal.GetFile(path)
		if err != nil {
			panic(err)
		}
		err = yaml.Unmarshal(contentBuffer.Bytes(), &data)
		if err != nil {
			panic(err)
		}
	}
	yamlOut, err := yaml.Marshal(data)
	if err != nil {
		panic(err)
	}
	logger.Debug("Data: " + string(yamlOut))
	if stdout {
		_, err := os.Stdout.Write(yamlOut)
		if err != nil {
			panic(err)
		}
	}
}

func validateSettings(settings CLIOptions) {
	var err error
	var errs []error
	var code, v int64

	if settings.Stdout && len(settings.TemplateFiles) > 1 {
		err = errors.New("cannot use `stdout` with multiple template files")
		v, _ = strconv.ParseInt("1", 2, 64)
		code += v
		errs = append(errs, err)
	}
	if settings.Stdin && len(settings.DataFiles) != 0 {
		err = errors.New("cannot use `stdin` with data files")
		v, _ = strconv.ParseInt("10", 2, 64)
		code += v
		errs = append(errs, err)
	}
	if len(settings.TemplateFiles) == 0 {
		err = errors.New("must provide at least one template file")
		v, _ = strconv.ParseInt("100", 2, 64)
		code += v
		errs = append(errs, err)
	}

	if err != nil {
		for _, err := range errs {
			logger.Error(err.Error())
		}
		os.Exit(int(code))
	}
}
