package main

import (
	"github.com/adam-huganir/yutc/internal"
	"os"
)

var YutcLog = &internal.YutcLog

func init() {
	internal.InitLogger("")
	YutcLog.Trace().Msg("yutc.init() called")

	rootCommand.Flags().SortFlags = false
	rootCommand.Flags().StringArrayVarP(
		&runSettings.DataFiles,
		"data",
		"d",
		nil,
		"Data file to parse and merge. Can be a file or a URL. "+
			"Can be specified multiple times and the inputs will be merged.",
	)
	rootCommand.Flags().StringArrayVarP(
		&runSettings.CommonTemplateFiles,
		"common-templates",
		"c",
		nil,
		"Templates to be shared across all arguments in template list. Can be a file or a URL. Can be specified multiple times.",
	)
	rootCommand.Flags().StringVarP(&runSettings.Output, "output", "o", "-", "Output file/directory, defaults to stdout")
	rootCommand.Flags().BoolVarP(&runSettings.Overwrite, "overwrite", "w", false, "Overwrite existing files")
	rootCommand.Flags().BoolVar(&runSettings.Version, "version", false, "Print the version and exit")

	rootCommand.Flags().BoolVarP(&runSettings.Recursive, "recursive", "r", false, "Recursively process directories")
	rootCommand.Flags().StringArrayVar(&runSettings.ExcludePatterns, "exclude", nil, "Exclude files matching the pattern")
	rootCommand.Flags().StringArrayVar(&runSettings.IncludePatterns, "include", nil, "Include files matching the pattern")

	rootCommand.PersistentFlags().BoolVarP(&runSettings.Verbose, "verbose", "v", false, "Verbose output")
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
