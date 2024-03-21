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

	// some things handled by cobra:
	// - min required args
	// - general type validation
	// - mutually exclusive flags (sometimes, i may handle them here for better error logging)

	outputFiles := settings.Output != "-"
	if !outputFiles && (len(settings.TemplateFiles) > 1 || settings.Recursive) {
		err = errors.New("cannot use `stdout` with multiple template files or --recursive flag")
		v, _ = strconv.ParseInt("1", 2, 64)
		code += v
		errs = append(errs, err)
	}
	if outputFiles {
		_, err = os.Stat(settings.Output)
		if err != nil {
			if os.IsNotExist(err) && len(settings.TemplateFiles) > 1 {
				err = errors.New("folder " + settings.Output + " does not exist to generate multiple templates")
				v, _ = strconv.ParseInt("10", 2, 64)
				code += v
				errs = append(errs, err)
			}
		} else {
			if !settings.Overwrite && len(settings.TemplateFiles) == 1 {
				err = errors.New("file " + settings.Output + " exists and `overwrite` is not set")
				v, _ = strconv.ParseInt("100", 2, 64)
				code += v
				errs = append(errs, err)
			}
		}
	}

	if settings.Overwrite && !outputFiles {
		err = errors.New("cannot use `overwrite` with `stdout`")
		v, _ = strconv.ParseInt("1000", 2, 64)
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
		v, _ = strconv.ParseInt("10000", 2, 64)
		code += v
		errs = append(errs, err)
	}

	missingFiles := false
	for _, f := range slices.Concat(settings.DataFiles, settings.CommonTemplateFiles, settings.TemplateFiles) {
		if f == "-" {
			continue
		}
		_, err = os.Stat(f)
		if err != nil {
			if os.IsNotExist(err) {
				err = errors.New("input file " + f + " does not exist")
				if !missingFiles {
					v, _ = strconv.ParseInt(
						"100000", 2, 64,
					)
				}
				missingFiles = true
				code += v
				errs = append(errs, err)
			}
		}
	}

	// going to be a bit overly strict with the recursive flag here, probably going to relax it later
	// like posix commands, but for now it is strict
	for _, templateFile := range settings.TemplateFiles {
		info, err := os.Stat(templateFile)
		if err != nil {
			continue // handled in previous check
		}
		if !info.IsDir() && settings.Recursive {
			err = errors.New("template file " + templateFile + " is not a directory, yet --recursive flag is set")
			v, _ = strconv.ParseInt("1000000", 2, 64)
			code += v
			errs = append(errs, err)
		} else if info.IsDir() && !settings.Recursive {
			err = errors.New("template file " + templateFile + " is a directory, yet --recursive flag is not set")
			v, _ = strconv.ParseInt("1000000", 2, 64)
			code += v
			errs = append(errs, err)

		}
	}

	// mutually exclusive flags
	if settings.IncludePatterns != nil && settings.ExcludePatterns != nil {
		err = errors.New("cannot use both --include and --exclude patterns")
		v, _ = strconv.ParseInt("10000000", 2, 64)
		code += v
		errs = append(errs, err)
	}

	if !settings.Recursive && (settings.IncludePatterns != nil || settings.ExcludePatterns != nil) {
		err = errors.New("cannot use include or exclude patterns without --recursive flag")
		v, _ = strconv.ParseInt("10000000", 2, 64)
		code += v
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		for _, err := range errs {
			YutcLog.Error().Err(err).Msg("argument validation error")
		}
	}
	return code
}

type CLISettings struct {
	DataFiles           []string `json:"data-files"`
	CommonTemplateFiles []string `json:"common-templates"`
	TemplateFiles       []string `json:"template-files"`

	Output    string `json:"output"`
	Overwrite bool   `json:"overwrite"`

	Recursive       bool     `json:"recursive"`
	ExcludePatterns []string `json:"exclude"`
	IncludePatterns []string `json:"include"`
	Version         bool     `json:"version"`

	Verbose bool `json:"verbose"`
}
