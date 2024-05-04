package internal

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
)

var ExitCode = new(int)

// YutcSettings is a struct to hold all the settings from the CLI
type YutcSettings struct {
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

	BearerToken      string `json:"bearer-auth"`
	BasicAuth        string `json:"basic-auth"`
	IncludeFilenames bool   `json:"include-filenames"`
}

func NewCLISettings() *YutcSettings {
	return &YutcSettings{}
}

func mustParseInt(binaryRep string) int {
	i, err := strconv.ParseInt(binaryRep, 2, 64)
	if err != nil {
		panic(err)
	}
	return int(i)
}

var ExitCodeMap = map[string]int{
	"ok":                         mustParseInt("0"), // 0
	"output file is a directory": mustParseInt("1"), // 1
	"cannot use `stdout` with multiple template files": mustParseInt("10"),      // 2
	"file exists and `overwrite` is not set":           mustParseInt("100"),     // 4
	"cannot use stdin with multiple files":             mustParseInt("1000"),    // 8
	"cannot use `overwrite` with `stdout`":             mustParseInt("10000"),   // 16
	"input file does not exist":                        mustParseInt("100000"),  // 32
	"cannot use both a pattern match and file input":   mustParseInt("1000000"), // 64
}

// ValidateArguments checks the arguments for the CLI and returns a code for the error
func ValidateArguments(
	settings *YutcSettings,
) (code int, errs []error) {
	var err error

	// some things handled by cobra:
	// - min required args
	// - general type validation
	// - mutually exclusive flags (sometimes, i may handle them here for better error logging)

	code, errs = validateOutput(settings, code, errs)
	code, errs = validateStdin(settings, code, errs)
	code, errs = verifyFilesExist(settings, code, errs)
	code, errs = verifyMutuallyExclusives(settings, code, errs)

	if len(errs) > 0 {
		YutcLog.Debug().Msg(fmt.Sprintf("Errors found: %d", len(errs)))
		for _, err = range errs {
			YutcLog.Error().Err(err).Msg("argument validation error")
		}
	}
	return code, errs
}

// verifyMutuallyExclusives checks for mutually exclusive flags
func verifyMutuallyExclusives(settings *YutcSettings, code int, errs []error) (int, []error) {
	var err error

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
			code += ExitCodeMap["cannot use both a pattern match and file input"]
			errs = append(errs, err)
		}
	}
	return code, errs
}

// verifyFilesExist checks that all the input files exist
func verifyFilesExist(settings *YutcSettings, code int, errs []error) (int, []error) {
	missingFiles := false

	for _, f := range slices.Concat(settings.DataFiles, settings.CommonTemplateFiles, settings.TemplatePaths) {
		if f == "-" {
			continue
		}
		_, err := os.Stat(f)
		if err != nil {
			if os.IsNotExist(err) {
				err = errors.New("input file " + f + " does not exist")
				if !missingFiles {
					code += ExitCodeMap["input file does not exist"]
				}
				missingFiles = true
				errs = append(errs, err)
			}
		}
	}
	return code, errs
}

// validateStdin checks if stdin is used in multiple places (which is a no no)
func validateStdin(settings *YutcSettings, code int, errs []error) (int, []error) {
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
		err := errors.New("cannot use stdin with multiple template or data files")
		code += ExitCodeMap["cannot use stdin with multiple files"]
		errs = append(errs, err)
	}
	return code, errs
}

// validateOutput checks if the output file exists and if it should be overwritten
func validateOutput(settings *YutcSettings, code int, errs []error) (int, []error) {
	var err error
	var outputFiles bool

	outputFiles = settings.Output != "-"
	if settings.Overwrite && !outputFiles {
		err = errors.New("cannot use `overwrite` with `stdout`")
		code += ExitCodeMap["cannot use `overwrite` with `stdout`"]
		errs = append(errs, err)
	}
	if !outputFiles && len(settings.TemplatePaths) > 1 {
		err = errors.New("cannot use `stdout` with multiple template files flag")
		code += ExitCodeMap["cannot use `stdout` with multiple template files"]
		errs = append(errs, err)
	}
	if outputFiles {
		_, err = os.Stat(settings.Output)
		if err != nil {
			if os.IsNotExist(err) && len(settings.TemplatePaths) > 1 {
				YutcLog.Debug().Msg(fmt.Sprintf("Directory does not exist, we will create: '%s'", settings.Output))
			}
		} else {
			if !settings.Overwrite && len(settings.TemplatePaths) == 1 {
				err = errors.New("file " + settings.Output + " exists and `overwrite` is not set")
				code += ExitCodeMap["file exists and `overwrite` is not set"]
				errs = append(errs, err)
			}
		}
	}
	return code, errs
}
