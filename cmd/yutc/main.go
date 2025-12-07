// yutc is a command-line tool for generating files from templates.
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
	"github.com/spf13/cobra"
)

var logger zerolog.Logger

func init() {
	logger = logging.InitLogger("")
	logger.Trace().Msg("main.init() called")
}

func initRoot(rootCommand *cobra.Command, runSettings *types.Arguments) {
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
	rootCommand.Flags().StringArrayVarP(
		&runSettings.SetData,
		"set",
		"",
		nil,
		"Set a data value via a key path. Can be specified multiple times.",
	)
	rootCommand.Flags().StringArrayVarP(
		&runSettings.CommonTemplateFiles,
		"common-templates",
		"c",
		nil,
		"Templates to be shared across all arguments in template list. Can be a file or a URL. "+
			"Can be specified multiple times.",
	)

	rootCommand.Flags().StringVarP(&runSettings.Output, "output", "o", "-", "Output file/directory, defaults to stdout")

	rootCommand.Flags().BoolVarP(&runSettings.IgnoreEmpty, "ignore-empty", "", false, "Do not copy empty files to output location")

	rootCommand.Flags().BoolVar(&runSettings.IncludeFilenames, "include-filenames", false, "Process filenames as templates")
	rootCommand.Flags().BoolVar(&runSettings.Strict, "strict", false, "On missing value, throw error instead of zero")
	rootCommand.Flags().BoolVarP(&runSettings.Overwrite, "overwrite", "w", false, "Overwrite existing files")
	rootCommand.Flags().BoolVar(&runSettings.Helm, "helm", false, "Enable Helm-specific data processing (Convert keys specified with key=Chart to pascalcase)")

	rootCommand.Flags().StringVar(&runSettings.BearerToken, "bearer-auth", "", "Bearer token for any URL authentication")
	rootCommand.Flags().StringVar(&runSettings.BasicAuth, "basic-auth", "", "Basic auth for any URL authentication")

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
	runData := &types.RunData{}
	rootCommand := newRootCommand(settings, runData, &logger)
	initRoot(rootCommand, settings)

	err := rootCommand.ExecuteContext(ctx)
	if err != nil {
		var exitErr *types.ExitError
		if errors.As(err, &exitErr) {
			logger.Error().Msg(exitErr.Error())
		}
	}
}
