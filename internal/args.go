package internal

import (
	"errors"
	"os"
	"slices"
	"strconv"
)

func ValidateArguments(
	settings *CLISettings,
) int64 {
	var err error
	var errs []error
	var code, v int64

	if len(settings.TemplateFiles) == 0 {
		err = errors.New("must provide at least one template file")
		v, _ = strconv.ParseInt("1", 2, 64)
		code += v
		errs = append(errs, err)
	}

	outputFiles := settings.Output != "-"
	if !outputFiles && len(settings.TemplateFiles) > 1 {
		err = errors.New("cannot use `stdout` with multiple template files")
		v, _ = strconv.ParseInt("100", 2, 64)
		code += v
		errs = append(errs, err)
	}
	if outputFiles {
		_, err = os.Stat(settings.Output)
		if err != nil {
			if os.IsNotExist(err) && len(settings.TemplateFiles) > 1 {
				err = errors.New("folder " + settings.Output + " does not exist to generate multiple templates")
				v, _ = strconv.ParseInt("1000", 2, 64)
				code += v
				errs = append(errs, err)
			}
		} else {
			if !settings.Overwrite && len(settings.TemplateFiles) == 1 {
				err = errors.New("file " + settings.Output + " exists and `overwrite` is not set")
				v, _ = strconv.ParseInt("10000", 2, 64)
				code += v
				errs = append(errs, err)
			}
		}
	}

	if settings.Overwrite && !outputFiles {
		err = errors.New("cannot use `overwrite` with `stdout`")
		v, _ = strconv.ParseInt("100000", 2, 64)
		code += v
		errs = append(errs, err)
	}

	stdins := 0
	for _, dataFile := range settings.DataFiles {
		if dataFile == "-" {
			stdins++
		}
	}
	for _, commonTemplate := range settings.CommonTemplateFiles {
		if commonTemplate == "-" {
			stdins++
		}
	}
	for _, templateFile := range settings.TemplateFiles {
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

	for _, f := range slices.Concat(settings.DataFiles, settings.CommonTemplateFiles, settings.TemplateFiles) {
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

type CLISettings struct {
	DataFiles           []string
	CommonTemplateFiles []string
	TemplateFiles       []string
	Output              string
	Overwrite           bool
	Version             bool
}
