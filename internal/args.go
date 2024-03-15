package internal

import (
	"errors"
	"os"
	"slices"
	"strconv"
)

func ValidateArguments(
	dataFiles, commonTemplateFiles, templateFiles []string,
	output string,
	overwrite bool,
) int64 {
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
	if outputFiles {
		_, err = os.Stat(output)
		if err != nil {
			if os.IsNotExist(err) && len(templateFiles) > 1 {
				err = errors.New("folder " + output + " does not exist to generate multiple templates")
				v, _ = strconv.ParseInt("1000", 2, 64)
				code += v
				errs = append(errs, err)
			}
		} else {
			if !overwrite && len(templateFiles) == 1 {
				err = errors.New("file " + output + " exists and `overwrite` is not set")
				v, _ = strconv.ParseInt("10000", 2, 64)
				code += v
				errs = append(errs, err)
			}
		}
	}

	if overwrite && !outputFiles {
		err = errors.New("cannot use `overwrite` with `stdout`")
		v, _ = strconv.ParseInt("100000", 2, 64)
		code += v
		errs = append(errs, err)
	}

	stdins := 0
	for _, dataFile := range dataFiles {
		if dataFile == "-" {
			stdins++
		}
	}
	for _, commonTemplate := range commonTemplateFiles {
		if commonTemplate == "-" {
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
		v, _ = strconv.ParseInt("1000000", 2, 64)
		code += v
		errs = append(errs, err)
	}

	for _, f := range slices.Concat(dataFiles, commonTemplateFiles, templateFiles) {
		if f == "-" {
			continue
		}
		_, err = os.Stat(f)
		if err != nil {
			if os.IsNotExist(err) {
				err = errors.New("input file " + f + " does not exist")
				v, _ = strconv.ParseInt("10000000", 2, 64)
				code += v
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		for _, err := range errs {
			YutcLog.Error().Err(err)
		}
	}
	return code
}
