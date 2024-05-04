package main

import (
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
	"os"
)

var runSettings *internal.YutcSettings
var YutcLog = &internal.YutcLog

func init() {
	internal.InitLogger("")
	YutcLog.Trace().Msg("yutc.init() called")
}

func initRoot(rootCommand *cobra.Command, settings *internal.YutcSettings) {
	const matchMessage = "Regex patterns to match/exclude from. A `!` prefix will exclude the pattern. Implies a recursive search."

	rootCommand.Flags().SortFlags = false
	rootCommand.Flags().StringArrayVarP(
		&settings.DataFiles,
		"data",
		"d",
		nil,
		"Data file to parse and merge. Can be a file or a URL. "+
			"Can be specified multiple times and the inputs will be merged.",
	)
	rootCommand.Flags().StringArrayVar(&settings.DataMatch, "data-match", nil, matchMessage)
	rootCommand.Flags().StringArrayVarP(
		&settings.CommonTemplateFiles,
		"common-templates",
		"c",
		nil,
		"Templates to be shared across all arguments in template list. Can be a file or a URL. "+
			"Can be specified multiple times.",
	)
	rootCommand.Flags().StringArrayVar(&settings.CommonTemplateMatch, "common-match", nil, matchMessage)
	rootCommand.Flags().StringVarP(&settings.Output, "output", "o", "-", "Output file/directory, defaults to stdout")
	rootCommand.Flags().BoolVarP(&settings.Overwrite, "overwrite", "w", false, "Overwrite existing files")
	rootCommand.Flags().BoolVar(&settings.IncludeFilenames, "include-filenames", false, "Exec any filenames with go templates")
	rootCommand.Flags().StringVar(&settings.BearerToken, "bearer-auth", "", "Bearer token for any URL authentication")
	rootCommand.Flags().StringVar(&settings.BasicAuth, "basic-auth", "", "Basic auth for any URL authentication")

	rootCommand.Flags().StringArrayVarP(
		&settings.TemplateMatch,
		"match",
		"m",
		nil,
		"For template arguments input, "+matchMessage,
	)
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
	YutcLog.Trace().Msg("yutc.main() called, executing rootCommand")
	rootCommand := newRootCommand()
	runSettings = internal.NewCLISettings()
	initRoot(rootCommand, runSettings)
	err := rootCommand.Execute()
	if err != nil {
		YutcLog.Error().Msg(err.Error())
		if exitCode == 0 {
			exitCode = -1
		}
	}
	os.Exit(exitCode)
}
