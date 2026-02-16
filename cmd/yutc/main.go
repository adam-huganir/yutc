// yutc is a command-line tool for generating data from templates.
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/adam-huganir/yutc/pkg"
	"github.com/adam-huganir/yutc/pkg/config"
	"github.com/adam-huganir/yutc/pkg/logging"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var logger zerolog.Logger

func init() {
	logger = logging.InitLogger("")
	logger.Trace().Msg("main.init() called")
}

func initRoot(rootCommand *cobra.Command, runSettings *types.Arguments) {
	rootCommand.Flags().SortFlags = false

	// Define groups
	dataTemplateGroup := pflag.NewFlagSet("Data & Templates", pflag.ContinueOnError)
	outputGroup := pflag.NewFlagSet("Output & Rendering", pflag.ContinueOnError)
	systemGroup := pflag.NewFlagSet("System", pflag.ContinueOnError)

	// Data & Templates
	dataTemplateGroup.StringArrayVarP(
		&runSettings.DataFiles,
		"data",
		"d",
		nil,
		"Data file to parse and merge. Can be a file or a URL. "+
			"Can be specified multiple times and the inputs will be merged. "+
			"Optionally nest data under a top-level key using: jsonpath=<path>,src=<path>  "+
			"See --help=syntax for more details.",
	)
	dataTemplateGroup.StringArrayVarP(
		&runSettings.SetData,
		"set",
		"",
		nil,
		"Set a data value via a key path. Can be specified multiple times.",
	)
	dataTemplateGroup.BoolVar(&runSettings.Helm, "helm", false, "Enable Helm-specific data processing (Convert keys specified with key=Chart to pascalcase)")

	dataTemplateGroup.StringArrayVarP(
		&runSettings.CommonTemplateFiles,
		"common-templates",
		"c",
		nil,
		"Templates to be shared across all arguments in template list. Can be a file or a URL. "+
			"Can be specified multiple times.",
	)
	dataTemplateGroup.BoolVar(&runSettings.IncludeFilenames, "include-filenames", false, "Process filenames as templates")

	// Global Auth for any URL source
	dataTemplateGroup.StringVar(&runSettings.Auth, "auth", "", "Authentication for any URL source. Format: 'user:pass' for Basic Auth or 'token' for Bearer Token.")

	// Output & Rendering
	outputGroup.StringVarP(&runSettings.Output, "output", "o", "-", "Output file/directory, defaults to stdout")
	outputGroup.BoolVarP(&runSettings.Overwrite, "overwrite", "w", false, "Overwrite existing files")
	outputGroup.BoolVarP(&runSettings.IgnoreEmpty, "ignore-empty", "", false, "Skip writing empty rendered template output to output location")
	outputGroup.BoolVar(&runSettings.Strict, "strict", false, "On missing value, throw error instead of zero")
	outputGroup.StringVar(&runSettings.DropExtension, "drop-extension", "tmpl", "Drop file extension from output filename before outputting")

	// Meta
	systemGroup.BoolVarP(
		&runSettings.Verbose,
		"verbose",
		"v",
		false,
		"Verbose output",
	)
	systemGroup.BoolVar(&runSettings.Version, "version", false, "Print the version and exit")

	// Add groups to root command
	rootCommand.Flags().AddFlagSet(dataTemplateGroup)
	rootCommand.Flags().AddFlagSet(outputGroup)
	rootCommand.PersistentFlags().AddFlagSet(systemGroup)

	// Configure help with groups
	ConfigureHelp(rootCommand, []*pflag.FlagSet{dataTemplateGroup, outputGroup, systemGroup})
	// Add help flag to system group
	if h := rootCommand.Flags().Lookup("help"); h != nil {
		systemGroup.AddFlag(h)
	}

}

func main() {
	logger.Trace().Msg("main.main() called")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	settings := config.NewCLISettings()
	runData := &yutc.RunData{}
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
