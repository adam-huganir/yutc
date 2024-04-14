package internal

import (
	"errors"
	"os"
	"slices"
	"strconv"
)

// ValidateArguments checks the arguments for the CLI and returns a code for the error
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
	if !outputFiles && len(settings.TemplatePaths) > 1 {
		err = errors.New("cannot use `stdout` with multiple template files flag")
		v, _ = strconv.ParseInt("1", 2, 64)
		code += v
		errs = append(errs, err)
	}
	if outputFiles {
		_, err = os.Stat(settings.Output)
		if err != nil {
			if os.IsNotExist(err) && len(settings.TemplatePaths) > 1 {
				err = errors.New("folder " + settings.Output + " does not exist to generate multiple templates")
				v, _ = strconv.ParseInt("10", 2, 64)
				code += v
				errs = append(errs, err)
			}
		} else {
			if !settings.Overwrite && len(settings.TemplatePaths) == 1 {
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
	for _, templateFile := range settings.TemplatePaths {
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
	for _, f := range slices.Concat(settings.DataFiles, settings.CommonTemplateFiles, settings.TemplatePaths) {
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

	// mutually exclusive flags
	if settings.TemplateMatch != nil {
		inputFiles := 0
		for _, templateFile := range settings.TemplatePaths {
			isDir, err := CheckIfDir(templateFile)
			if err != nil {
				continue
			}
			if !*isDir {
				inputFiles++
			}
		}
		if inputFiles > 0 {
			err = errors.New("cannot use both a pattern match and a file input for templates, since a pattern match implies a recursive search")
			v, _ = strconv.ParseInt("10000000", 2, 64)
			code += v
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		for _, err := range errs {
			YutcLog.Error().Err(err).Msg("argument validation error")
		}
	}
	return code
}

// CLISettings is a struct to hold all the settings from the CLI
type CLISettings struct {
	DataFiles []string `json:"data-files"`
	DataMatch []string `json:"data-match"`

	CommonTemplateFiles []string `json:"common-templates"`
	CommonTemplateMatch []string `json:"common-templates-match"`

	TemplatePaths []string `json:"template-files"`
	TemplateMatch []string `json:"template-match"`

	Output    string `json:"output"`
	Overwrite bool   `json:"overwrite"`

	Version bool `json:"version"`
	Verbose bool `json:"verbose"`

	BearerToken string `json:"bearer-auth"`
	BasicAuth   string `json:"basic-auth"`
}

var RunSettings = &CLISettings{}
