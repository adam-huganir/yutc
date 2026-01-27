// Package config handles CLI argument parsing, validation, and configuration management.
package config

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/adam-huganir/yutc/pkg/data"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
)

// NewCLISettings creates and returns a new Arguments struct with default values.
func NewCLISettings() *types.Arguments {
	return &types.Arguments{}
}

// ValidateArguments checks the arguments for the CLI and returns a ValidationError if any issues are found
func ValidateArguments(arguments *types.Arguments, logger *zerolog.Logger) error {
	var err error
	var errs []error

	// some things handled by cobra:
	// - min required args
	// - general type validation
	// - mutually exclusive flags (sometimes, i may handle them here for better error logging)
	errs = validateOutput(arguments, errs, logger)
	errs = validateStructuredInput(arguments, errs)
	errs = validateStdin(arguments, errs)
	errs = verifyFilesExist(arguments, errs)
	errs = verifyMutuallyExclusives(arguments, errs)

	if len(errs) > 0 {
		logger.Debug().Msg(fmt.Sprintf("Errors found: %d", len(errs)))
		for _, err = range errs {
			logger.Error().Err(err).Msg("argument validation error")
		}
		return &types.ValidationError{Errors: errs}
	}
	return nil
}

func validateStructuredInput(args *types.Arguments, errs []error) []error {
	// if we are doing a folder or archive, it must be the _only_ specified input
	// other behavior is currently undefined and will error
	logger := zerolog.Nop()
	df, err := data.ParseFileArgs(args.DataFiles, "", &logger)
	if err != nil {
		return append(errs, err)
	}
	dataRecursables, err := data.CountRecursables(slices.Concat(df...))
	if err != nil {
		return append(errs, err)
	}
	ct, err := data.ParseFileArgs(args.CommonTemplateFiles, "", &logger)
	if err != nil {
		return append(errs, err)
	}
	commonRecursables, err := data.CountRecursables(slices.Concat(ct...))
	if err != nil {
		return append(errs, err)
	}
	tp, err := data.ParseFileArgs(args.TemplatePaths, "", &logger)
	if err != nil {
		return append(errs, err)
	}
	templateRecursables, err := data.CountRecursables(slices.Concat(tp...))
	if err != nil {
		return append(errs, err)
	}

	if dataRecursables > 1 && len(args.DataFiles) != dataRecursables ||
		commonRecursables > 1 && len(args.CommonTemplateFiles) != commonRecursables ||
		templateRecursables > 1 && len(args.TemplatePaths) != templateRecursables {
		err = errors.New("found both data and recursables as inputs")
		errs = append(errs, err)
	}

	return errs
}

// verifyMutuallyExclusives checks for mutually exclusive flags (currently a no-op)
func verifyMutuallyExclusives(_ *types.Arguments, errs []error) []error {
	return errs
}

// verifyFilesExist checks that all the input data exist
func verifyFilesExist(args *types.Arguments, errs []error) []error {
	// For data, we need to parse them to extract the actual path
	for _, dataFileArg := range args.DataFiles {
		dataArgs, err := data.ParseFileArg(dataFileArg, "")
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, dataArg := range dataArgs {
			f := dataArg.Name
			if f == "-" {
				continue
			}
			_, err = os.Stat(f)
			if err != nil {
				if os.IsNotExist(err) {
					err = errors.New("input file " + f + " does not exist")
					errs = append(errs, err)
				}
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
				errs = append(errs, err)
			}
		}
	}
	return errs
}

// validateStdin checks if stdin is used in multiple places (which is a no no)
func validateStdin(args *types.Arguments, errs []error) []error {
	nStdin := 0
	for _, dataFileArg := range args.DataFiles {
		dataArgs, err := data.ParseFileArg(dataFileArg, "")
		if err != nil {
			// Error will be caught in verifyFilesExist
			continue
		}
		for _, dataArg := range dataArgs {
			if dataArg.Name == "-" {
				nStdin++
			}
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
		err := errors.New("cannot use stdin with multiple template or data data")
		errs = append(errs, err)
	}
	return errs
}

// validateOutput checks if the output file exists and if it should be overwritten
func validateOutput(args *types.Arguments, errs []error, logger *zerolog.Logger) []error {
	var err error
	outputFiles := args.Output != "-"
	if args.Overwrite && !outputFiles {
		err = errors.New("cannot use `overwrite` with `stdout`")
		errs = append(errs, err)
	}
	if !outputFiles && len(args.TemplatePaths) > 1 {
		err = errors.New("cannot use `stdout` with multiple template data flag")
		errs = append(errs, err)
	}
	if outputFiles {
		isDir, err := data.IsDir(args.Output)
		if err != nil {
			if os.IsNotExist(err) && len(args.TemplatePaths) > 1 {
				logger.Debug().Msg(fmt.Sprintf("Directory does not exist, we will create: '%s'", args.Output))
			}
		} else if !isDir {
			if !args.Overwrite && len(args.TemplatePaths) == 1 {
				err = errors.New("file " + args.Output + " exists and `overwrite` is not set")
				errs = append(errs, err)
			}
		}
	}
	return errs
}
