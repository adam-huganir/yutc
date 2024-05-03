package main

import (
	"github.com/adam-huganir/yutc/internal"
	"os"
)

var YutcLog = &internal.YutcLog
var RunSettings *internal.CLISettings

func init() {
	internal.InitLogger("")
	YutcLog.Trace().Msg("yutc.init() called")

	const matchMessage = "Regex patterns to match/exclude from. A `!` prefix will exclude the pattern. Implies a recursive search."

	RunSettings = internal.NewCLISettings()
	rootCommand.Flags().SortFlags = false
	rootCommand.Flags().StringArrayVarP(
		&RunSettings.DataFiles,
		"data",
		"d",
		nil,
		"Data file to parse and merge. Can be a file or a URL. "+
			"Can be specified multiple times and the inputs will be merged.",
	)
	rootCommand.Flags().StringArrayVar(&RunSettings.DataMatch, "data-match", nil, matchMessage)
	rootCommand.Flags().StringArrayVarP(
		&RunSettings.CommonTemplateFiles,
		"common-templates",
		"c",
		nil,
		"Templates to be shared across all arguments in template list. Can be a file or a URL. "+
			"Can be specified multiple times.",
	)
	rootCommand.Flags().StringArrayVar(&RunSettings.CommonTemplateMatch, "common-match", nil, matchMessage)
	rootCommand.Flags().StringVarP(&RunSettings.Output, "output", "o", "-", "Output file/directory, defaults to stdout")
	rootCommand.Flags().BoolVarP(&RunSettings.Overwrite, "overwrite", "w", false, "Overwrite existing files")
	rootCommand.Flags().BoolVar(&RunSettings.IncludeFilenames, "include-filenames", false, "Exec any filenames with go templates")
	rootCommand.Flags().StringVar(&RunSettings.BearerToken, "bearer-auth", "", "Bearer token for any URL authentication")
	rootCommand.Flags().StringVar(&RunSettings.BasicAuth, "basic-auth", "", "Basic auth for any URL authentication")

	rootCommand.Flags().StringArrayVarP(
		&RunSettings.TemplateMatch,
		"match",
		"m",
		nil,
		"For template arguments input, "+matchMessage,
	)
	rootCommand.PersistentFlags().BoolVarP(
		&RunSettings.Verbose,
		"verbose",
		"v",
		false,
		"Verbose output",
	)
	rootCommand.Flags().BoolVar(&RunSettings.Version, "version", false, "Print the version and exit")

}

func main() {
	YutcLog.Trace().Msg("yutc.main() called, executing rootCommand")
	err := rootCommand.Execute()
	if err != nil {
		YutcLog.Error().Msg(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}
