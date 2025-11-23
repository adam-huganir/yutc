// Package config handles CLI argument parsing, validation, and configuration management.
package config

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"

	"github.com/adam-huganir/yutc/pkg/files"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
)

// ErrorMessage represents a validation error message.
type ErrorMessage string

// ExitCode represents a CLI exit code.
type ExitCode int

// NewCLISettings creates and returns a new Arguments struct with default values.
func NewCLISettings() *types.Arguments {
	return &types.Arguments{}
}

func mustParseInt(binaryRep string) int {
	i, err := strconv.ParseInt(binaryRep, 2, 64)
	if err != nil {
		panic(err)
	}
	return int(i)
}

// ExitCodeMap maps error messages to exit codes (TODO: remove this with something less clunky)
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
func ValidateArguments(arguments *types.Arguments, logger *zerolog.Logger) (code int, errs []error) {
	var err error

	// some things handled by cobra:
	// - min required args
	// - general type validation
	// - mutually exclusive flags (sometimes, i may handle them here for better error logging)
	code, errs = validateOutput(arguments, code, errs, *logger)
	code, errs = validateStructuredInput(arguments, code, errs)
	code, errs = validateStdin(arguments, code, errs)
	code, errs = verifyFilesExist(arguments, code, errs)
	code, errs = verifyMutuallyExclusives(arguments, code, errs)

	if len(errs) > 0 {
		logger.Debug().Msg(fmt.Sprintf("Errors found: %d", len(errs)))
		for _, err = range errs {
			logger.Error().Err(err).Msg("argument validation error")
		}
	}
	return code, errs
}

func validateStructuredInput(args *types.Arguments, code int, errs []error) (int, []error) {
	// if we are doing a folder or archive, it must be the _only_ specified input
	// other behavior is currently undefined and will error

	dataRecursables, err := files.CountDataRecursables(args.DataFiles)
	if err != nil {
		panic(err)
	}
	commonRecursables, err := files.CountRecursables(args.CommonTemplateFiles)
	if err != nil {
		panic(err)
	}
	templateRecursables, err := files.CountRecursables(args.TemplatePaths)
	if err != nil {
		panic(err)
	}

	if dataRecursables > 1 && len(args.DataFiles) != dataRecursables ||
		commonRecursables > 1 && len(args.CommonTemplateFiles) != commonRecursables ||
		templateRecursables > 1 && len(args.TemplatePaths) != templateRecursables {
		err = errors.New("found both files and recursables as inputs")
		code += ExitCodeMap["found both files and recursables as inputs"]
		errs = append(errs, err)
	}

	return code, errs
}

// verifyMutuallyExclusives checks for mutually exclusive flags (currently a no-op)
func verifyMutuallyExclusives(_ *types.Arguments, code int, errs []error) (int, []error) {
	return code, errs
}

// verifyFilesExist checks that all the input files exist
func verifyFilesExist(args *types.Arguments, code int, errs []error) (int, []error) {
	missingFiles := false
	// For data files, we need to parse them to extract the actual path
	for _, dataFileArg := range args.DataFiles {
		dataArg, err := files.ParseDataFileArg(dataFileArg)
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
	for _, f := range slices.Concat(args.CommonTemplateFiles, args.TemplatePaths) {
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
func validateStdin(args *types.Arguments, code int, errs []error) (int, []error) {
	nStdin := 0
	for _, dataFileArg := range args.DataFiles {
		dataArg, err := files.ParseDataFileArg(dataFileArg)
		if err != nil {
			// Error will be caught in verifyFilesExist
			continue
		}
		if dataArg.Path == "-" {
			nStdin++
		}
	}
	for _, commonTemplate := range args.CommonTemplateFiles {
		if commonTemplate == "-" {
			nStdin++
		}
	}
	for _, templateFile := range args.TemplatePaths {
		if templateFile == "-" {
			nStdin++
		}
	}
	if nStdin > 1 {
		err := errors.New("cannot use stdin with multiple template or data files")
		code += ExitCodeMap["cannot use stdin with multiple files"]
		errs = append(errs, err)
	}
	return code, errs
}

// validateOutput checks if the output file exists and if it should be overwritten
func validateOutput(args *types.Arguments, code int, errs []error, logger zerolog.Logger) (int, []error) {
	var err error
	outputFiles := args.Output != "-"
	if args.Overwrite && !outputFiles {
		err = errors.New("cannot use `overwrite` with `stdout`")
		code += ExitCodeMap["cannot use `overwrite` with `stdout`"]
		errs = append(errs, err)
	}
	if !outputFiles && len(args.TemplatePaths) > 1 {
		err = errors.New("cannot use `stdout` with multiple template files flag")
		code += ExitCodeMap["cannot use `stdout` with multiple template files"]
		errs = append(errs, err)
	}
	if outputFiles {
		isDir, err := files.IsDir(args.Output)
		if err != nil {
			if os.IsNotExist(err) && len(args.TemplatePaths) > 1 {
				logger.Debug().Msg(fmt.Sprintf("Directory does not exist, we will create: '%s'", args.Output))
			}
		} else if !isDir {
			if !args.Overwrite && len(args.TemplatePaths) == 1 {
				err = errors.New("file " + args.Output + " exists and `overwrite` is not set")
				code += ExitCodeMap["file exists and `overwrite` is not set"]
				errs = append(errs, err)
			}
		}
	}
	return code, errs
}
