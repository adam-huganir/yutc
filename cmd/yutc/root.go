package main

import (
	"github.com/adam-huganir/yutc/internal"
	"github.com/adam-huganir/yutc/internal/types"
	"github.com/spf13/cobra"
)

func newRootCommand(settings *types.YutcSettings) *cobra.Command {
	rootCommand := &cobra.Command{
		Use:   "yutc [flags] <template_files...>",
		Short: "Yet Another Universal Template Converter",
		Long:  `YUTC is a CLI tool for converting templates using YAML/JSON data.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoot(settings, args)
		},
	}
	return rootCommand
}

func runRoot(settings *types.YutcSettings, args []string) error {
	app := internal.NewApp(settings, logger)
	return app.Run(args)
}
