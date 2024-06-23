package main

import (
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
	"os"
)

var runSettings *internal.YutcSettings
var YutcLog = &internal.YutcLog
var tempDir string

func init() {
	internal.InitLogger("")
	YutcLog.Trace().Msg("yutc.init() called")

	// we ignore errors as we may not need this temp directory depending on inputs
	// we will catch any issues later in usage
	tempDir, _ = internal.GenerateTempDirName("yutc-*")
}

func initRoot(rootCommand *cobra.Command, settings *internal.YutcSettings) {
	YutcLog.Trace().Msg("yutc.initRoot() called")
	rootCommand.Flags().SortFlags = false

	rootCommand.PersistentFlags().BoolVarP(
		&settings.Verbose,
		"verbose",
		"v",
		false,
		"Verbose output",
	)
	rootCommand.Flags().BoolVar(&settings.Version, "version", false, "Print the version and exit")
}

func initCommon(cmd *cobra.Command, settings *internal.YutcSettings) {
	YutcLog.Trace().Msg("yutc.initCommon() called")
	cmd.Flags().SortFlags = false

	cmd.PersistentFlags().BoolVarP(
		&settings.Verbose,
		"verbose",
		"v",
		false,
		"Verbose output",
	)
	cmd.Flags().BoolVar(&settings.Version, "version", false, "Print the version and exit")
	cmd.Flags().StringArrayVarP(
		&settings.DataFiles,
		"data",
		"d",
		nil,
		"Data file to parse and merge. Can be a file or a URL. "+
			"Can be specified multiple times and the inputs will be merged.",
	)
	cmd.Flags().StringArrayVarP(
		&settings.CommonTemplateFiles,
		"common-templates",
		"c",
		nil,
		"Templates to be shared across all arguments in template list. Can be a file or a URL. "+
			"Can be specified multiple times.",
	)

	cmd.Flags().StringVarP(&settings.Output, "output", "o", "-", "Output file/directory, defaults to stdout")

	cmd.Flags().BoolVar(&settings.IncludeFilenames, "include-filenames", false, "Exec any filenames with go templates")
	cmd.Flags().BoolVarP(&settings.Overwrite, "overwrite", "w", false, "Overwrite existing files")

	// probably a unnecessary feature but still rad. maybe add b64 encoding at some point
	cmd.Flags().StringVar(&settings.BearerToken, "bearer-auth", "", "Bearer token for any URL authentication")
	cmd.Flags().StringVar(&settings.BasicAuth, "basic-auth", "", "Basic auth for any URL authentication")

}

func initForEachCommand(cmd *cobra.Command, settings *internal.YutcSettings) {
	cmd.Flags().BoolVar(&settings.Append, "append", false, "Append data to output")
}

func main() {
	YutcLog.Trace().Msg("yutc.main() called, executing rootCommand")
	rootCommand := initCli()
	err := rootCommand.Execute()
	if err != nil {
		YutcLog.Error().Msg(err.Error())
		if *internal.ExitCode == 0 {
			*internal.ExitCode = -1
		}
	}
	os.Exit(*internal.ExitCode)
}

func initCli(settings ...*internal.YutcSettings) *cobra.Command {
	rootCommand := newRootCommand()
	templateCommand := newTemplateCommand()
	forEachCommand := newForEachCommand()
	rootCommand.AddCommand(templateCommand)
	rootCommand.AddCommand(forEachCommand)
	if len(settings) == 1 {
		runSettings = settings[0]
	} else if len(settings) == 0 {
		runSettings = internal.NewCLISettings()
	} else {
		YutcLog.Fatal().Msg("Too many settings passed to initCli")
	}
	initRoot(rootCommand, runSettings)
	initCommon(templateCommand, runSettings)
	return rootCommand
}
