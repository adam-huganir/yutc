package main

import (
	"errors"
	"os"

	"github.com/adam-huganir/yutc/internal/config"
	"github.com/adam-huganir/yutc/internal/files"
	"github.com/adam-huganir/yutc/internal/logging"
	"github.com/adam-huganir/yutc/internal/types"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var logger zerolog.Logger
var tempDir string

func init() {
	logger = logging.InitLogger("")
	logger.Trace().Msg("yutc.init() called")

	// we ignore errors as we may not need this temp directory depending on inputs
	// we will catch any issues later in usage
	tempDir, _ = files.GenerateTempDirName("yutc-*")
}

func initRoot(rootCommand *cobra.Command, settings *types.YutcSettings) {
	//const matchMessage = "Regex patterns to match/exclude from. A `!` prefix will exclude the pattern. Implies a recursive search."

	rootCommand.Flags().SortFlags = false
	rootCommand.Flags().StringArrayVarP(
		&settings.DataFiles,
		"data",
		"d",
		nil,
		"Data file to parse and merge. Can be a file or a URL. "+
			"Can be specified multiple times and the inputs will be merged. "+
			"Optionally nest data under a top-level key using: key=<name>,src=<path>",
	)
	//rootCommand.Flags().StringArrayVar(&settings.DataMatch, "data-match", nil, matchMessage)
	rootCommand.Flags().StringArrayVarP(
		&settings.CommonTemplateFiles,
		"common-templates",
		"c",
		nil,
		"Templates to be shared across all arguments in template list. Can be a file or a URL. "+
			"Can be specified multiple times.",
	)
	//rootCommand.Flags().StringArrayVar(&settings.CommonTemplateMatch, "common-match", nil, matchMessage)

	rootCommand.Flags().StringVarP(&settings.Output, "output", "o", "-", "Output file/directory, defaults to stdout")

	rootCommand.Flags().BoolVar(&settings.IncludeFilenames, "include-filenames", false, "Exec any filenames with go templates")
	rootCommand.Flags().BoolVar(&settings.Strict, "strict", false, "On missing value, throw error instead of zero")
	rootCommand.Flags().BoolVarP(&settings.Overwrite, "overwrite", "w", false, "Overwrite existing files")

	rootCommand.Flags().StringVar(&settings.BearerToken, "bearer-auth", "", "Bearer token for any URL authentication")
	rootCommand.Flags().StringVar(&settings.BasicAuth, "basic-auth", "", "Basic auth for any URL authentication")

	//rootCommand.Flags().StringArrayVarP(
	//	&settings.TemplateMatch,
	//	"match",
	//	"m",
	//	nil,
	//	"For template arguments input, "+matchMessage,
	//)
	rootCommand.PersistentFlags().BoolVarP(
		&settings.Verbose,
		"verbose",
		"v",
		false,
		"Verbose output",
	)
	rootCommand.Flags().BoolVar(&settings.Version, "version", false, "Print the version and exit")
}

func main() {
	logger.Trace().Msg("yutc.main() called, executing rootCommand")
	runSettings := config.NewCLISettings()
	rootCommand := newRootCommand(runSettings)
	initRoot(rootCommand, runSettings)

	err := rootCommand.Execute()
	if err != nil {
		var exitErr *types.ExitError
		if errors.As(err, &exitErr) {
			logger.Error().Msg(exitErr.Error())
			os.Exit(exitErr.Code)
		}
		logger.Error().Msg(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}
