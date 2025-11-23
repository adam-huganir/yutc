package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/adam-huganir/yutc/pkg/config"
	"github.com/adam-huganir/yutc/pkg/logging"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
)

var logger zerolog.Logger

func init() {
	logger = logging.InitLogger("")
	logger.Trace().Msg("main.init() called")
}

func initRoot(ctx context.Context) {
	//const matchMessage = "Regex patterns to match/exclude from. A `!` prefix will exclude the pattern. Implies a recursive search."
	rootCommand := config.GetCommand(ctx)
	runSettings := config.GetSettings(ctx)

	rootCommand.Flags().SortFlags = false
	rootCommand.Flags().StringArrayVarP(
		&runSettings.DataFiles,
		"data",
		"d",
		nil,
		"Data file to parse and merge. Can be a file or a URL. "+
			"Can be specified multiple times and the inputs will be merged. "+
			"Optionally nest data under a top-level key using: key=<name>,src=<path>",
	)
	//rootCommand.Flags().StringArrayVar(&runSettings.DataMatch, "data-match", nil, matchMessage)
	rootCommand.Flags().StringArrayVarP(
		&runSettings.CommonTemplateFiles,
		"common-templates",
		"c",
		nil,
		"Templates to be shared across all arguments in template list. Can be a file or a URL. "+
			"Can be specified multiple times.",
	)
	//rootCommand.Flags().StringArrayVar(&runSettings.CommonTemplateMatch, "common-match", nil, matchMessage)

	rootCommand.Flags().StringVarP(&runSettings.Output, "output", "o", "-", "Output file/directory, defaults to stdout")

	rootCommand.Flags().BoolVar(&runSettings.IncludeFilenames, "include-filenames", false, "Exec any filenames with go templates")
	rootCommand.Flags().BoolVar(&runSettings.Strict, "strict", false, "On missing value, throw error instead of zero")
	rootCommand.Flags().BoolVarP(&runSettings.Overwrite, "overwrite", "w", false, "Overwrite existing files")

	rootCommand.Flags().StringVar(&runSettings.BearerToken, "bearer-auth", "", "Bearer token for any URL authentication")
	rootCommand.Flags().StringVar(&runSettings.BasicAuth, "basic-auth", "", "Basic auth for any URL authentication")

	//rootCommand.Flags().StringArrayVarP(
	//	&runSettings.TemplateMatch,
	//	"match",
	//	"m",
	//	nil,
	//	"For template arguments input, "+matchMessage,
	//)
	rootCommand.PersistentFlags().BoolVarP(
		&runSettings.Verbose,
		"verbose",
		"v",
		false,
		"Verbose output",
	)
	rootCommand.Flags().BoolVar(&runSettings.Version, "version", false, "Print the version and exit")
}

func main() {
	logger.Trace().Msg("main.main() called")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	settings := config.NewCLISettings()
	rootCommand := newRootCommand(settings)
	// does not actually have an error case at this moment, but probably will at some point
	ctx, _ = config.LoadContext(ctx, rootCommand, settings, "", &logger)
	initRoot(ctx)

	err := rootCommand.ExecuteContext(ctx)
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
