package internal

import (
	"errors"
	"os"
	"strconv"
)

func ValidateArguments(
	dataFiles, sharedTemplates, templateFiles []string,
	output string,
	overwrite bool,
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

	outputFiles := output != "-"
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

	stdins := 0
	for _, dataFile := range dataFiles {
		if dataFile == "-" {
			stdins++
		}
	}
	for _, sharedTemplate := range sharedTemplates {
		if sharedTemplate == "-" {
			stdins++
		}
	}
	for _, templateFile := range templateFiles {
		if templateFile == "-" {
			stdins++
		}
	}
	if stdins > 1 {
		err = errors.New("cannot use stdin with multiple template or data files")
		v, _ = strconv.ParseInt("10000", 2, 64)
	}

	if len(errs) > 0 {
		for _, err := range errs {
			logger.Error(err.Error())
		}
		os.Exit(int(code))
	}
}
