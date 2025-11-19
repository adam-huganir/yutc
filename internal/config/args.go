package config

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"

	"github.com/adam-huganir/yutc/internal/data"
	"github.com/adam-huganir/yutc/internal/files"
	"github.com/adam-huganir/yutc/internal/logging"
	"github.com/adam-huganir/yutc/internal/types"
)

var ExitCode = new(int)

func NewCLISettings() *types.YutcSettings {
	return &types.YutcSettings{}
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
	"cannot use `stdout` with multiple template files": mustParseInt("10"),       // 2
	"file exists and `overwrite` is not set":           mustParseInt("100"),      // 4
	"cannot use stdin with multiple files":             mustParseInt("1000"),     // 8
	"cannot use `overwrite` with `stdout`":             mustParseInt("10000"),    // 16
	"input file does not exist":                        mustParseInt("100000"),   // 32
	"cannot use both a pattern match and file input":   mustParseInt("1000000"),  // 64
	"folder/tar files as inputs must be the only ones": mustParseInt("10000000"), // 64
}

// ValidateArguments checks the arguments for the CLI and returns a code for the error
func ValidateArguments(settings *types.YutcSettings) (code int, errs []error) {
	var err error

	// some things handled by cobra:
	// - min required args
	// - general type validation
	// - mutually exclusive flags (sometimes, i may handle them here for better error logging)
	code, errs = validateOutput(settings, code, errs)
	code, errs = validateStructuredInput(settings, code, errs)
	code, errs = validateStdin(settings, code, errs)
	code, errs = verifyFilesExist(settings, code, errs)
	code, errs = verifyMutuallyExclusives(settings, code, errs)

	if len(errs) > 0 {
		logging.YutcLog.Debug().Msg(fmt.Sprintf("Errors found: %d", len(errs)))
		for _, err = range errs {
			logging.YutcLog.Error().Err(err).Msg("argument validation error")
		}
	}
	return code, errs
}

func validateStructuredInput(settings *types.YutcSettings, code int, errs []error) (int, []error) {
	// if we are doing a folder or archive, it must be the _only_ specified input
	// other behavior is currently undefined and will error
	dataRecursables, err := data.CountDataRecursables(settings.DataFiles)
	if err != nil {
		panic(err)
	}
	commonRecursables, err := files.CountRecursables(settings.CommonTemplateFiles)
	if err != nil {
		panic(err)
	}
	templateRecursables, err := files.CountRecursables(settings.TemplatePaths)
	if err != nil {
		panic(err)
	}

	if dataRecursables > 1 && len(settings.DataFiles) != dataRecursables ||
		commonRecursables > 1 && len(settings.CommonTemplateFiles) != commonRecursables ||
		templateRecursables > 1 && len(settings.TemplatePaths) != templateRecursables {
		err = errors.New("found both files and recursables as inputs")
		code += ExitCodeMap["found both files and recursables as inputs"]
		errs = append(errs, err)
	}

	return code, errs
}

// verifyMutuallyExclusives checks for mutually exclusive flags
func verifyMutuallyExclusives(settings *types.YutcSettings, code int, errs []error) (int, []error) {
	return code, errs
}

// verifyFilesExist checks that all the input files exist
func verifyFilesExist(settings *types.YutcSettings, code int, errs []error) (int, []error) {
	missingFiles := false

	// For data files, we need to parse them to extract the actual path
	for _, dataFileArg := range settings.DataFiles {
		dataArg, err := data.ParseDataFileArg(dataFileArg)
		if err != nil {
			errs = append(errs, err)
			if !missingFiles {
				code += ExitCodeMap["input file does not exist"]
			}
			missingFiles = true
			continue
		}
		f := dataArg.Path
		if f == "-" {
			continue
		}
		_, err = os.Stat(f)
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

	// For common templates and template paths, check directly
	for _, f := range slices.Concat(settings.CommonTemplateFiles, settings.TemplatePaths) {
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
func validateStdin(settings *types.YutcSettings, code int, errs []error) (int, []error) {
	stdins := 0
	for _, dataFileArg := range settings.DataFiles {
		dataArg, err := data.ParseDataFileArg(dataFileArg)
		if err != nil {
			// Error will be caught in verifyFilesExist
			continue
		}
		if dataArg.Path == "-" {
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
func validateOutput(settings *types.YutcSettings, code int, errs []error) (int, []error) {
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
		isDir, err := files.IsDir(settings.Output)
		if err != nil {
			if os.IsNotExist(err) && len(settings.TemplatePaths) > 1 {
				logging.YutcLog.Debug().Msg(fmt.Sprintf("Directory does not exist, we will create: '%s'", settings.Output))
			}
		} else if !isDir {
			if !settings.Overwrite && len(settings.TemplatePaths) == 1 {
				err = errors.New("file " + settings.Output + " exists and `overwrite` is not set")
				code += ExitCodeMap["file exists and `overwrite` is not set"]
				errs = append(errs, err)
			}
		}
	}
	return code, errs
}

type RunData struct {
	*types.YutcSettings
	DataFiles []*types.DataFileArg
}

func (rd *RunData) ParseDataFiles() error {
	for _, dataFileArg := range rd.YutcSettings.DataFiles {
		dataArg, err := data.ParseDataFileArg(dataFileArg)
		if err != nil {
			return err
		}
		rd.DataFiles = append(rd.DataFiles, dataArg)
	}
	return nil
}
