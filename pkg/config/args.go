// Package config handles CLI argument parsing, validation, and configuration management.
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/adam-huganir/yutc/pkg/data"
	"github.com/adam-huganir/yutc/pkg/loader"
	"github.com/adam-huganir/yutc/pkg/templates"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
)

// ParsedInputs holds pre-parsed inputs so that validation does not need to re-parse raw strings.
type ParsedInputs struct {
	DataFiles           []*data.Input
	TemplateFiles       []*templates.Input
	CommonTemplateFiles []*templates.Input
}

// NewCLISettings creates and returns a new Arguments struct with default values.
func NewCLISettings() *types.Arguments {
	return &types.Arguments{}
}

// ValidateArguments checks the arguments for the CLI and returns a ValidationError if any issues are found.
// It accepts pre-parsed inputs so that arguments are only parsed once.
func ValidateArguments(arguments *types.Arguments, parsed *ParsedInputs, logger *zerolog.Logger) error {
	var err error
	var errs []error

	// some things handled by cobra:
	// - min required args
	// - general type validation
	// - mutually exclusive flags (sometimes, i may handle them here for better error logging)
	errs = validateOutput(arguments, errs, logger)
	errs = validateStructuredInput(arguments, parsed, errs)
	errs = validateStdin(parsed, errs)
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

func validateStructuredInput(args *types.Arguments, parsed *ParsedInputs, errs []error) []error {
	// if we are doing a folder or archive, it must be the _only_ specified input
	// other behavior is currently undefined and will error
	dataRecursables, err := data.CountDataRecursables(parsed.DataFiles)
	if err != nil {
		return append(errs, err)
	}
	commonRecursables, err := templates.CountTemplateRecursables(parsed.CommonTemplateFiles)
	if err != nil {
		return append(errs, err)
	}
	templateRecursables, err := templates.CountTemplateRecursables(parsed.TemplateFiles)
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

// validateStdin checks if stdin is used in multiple places (which is a no no)
func validateStdin(parsed *ParsedInputs, errs []error) []error {
	nStdin := 0
	for _, di := range parsed.DataFiles {
		if di.Name == "-" {
			nStdin++
		}
	}
	for _, ti := range parsed.CommonTemplateFiles {
		if ti.Name == "-" {
			nStdin++
		}
	}
	for _, ti := range parsed.TemplateFiles {
		if ti.Name == "-" {
			nStdin++
		}
	}
	if nStdin > 1 {
		err := errors.New("cannot use stdin with multiple template or data files")
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
		isDir, err := loader.IsDir(args.Output)
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
