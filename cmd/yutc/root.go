package main

import (
	"context"

	yutc "github.com/adam-huganir/yutc/pkg"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/spf13/cobra"
)

func newRootCommand(settings *types.Arguments) *cobra.Command {
	rootCommand := &cobra.Command{
		Use:   "yutc [flags] <template_files...>",
		Short: "Yet Another Universal Template Converter",
		Long:  `YUTC is a CLI tool for converting templates using YAML/JSON data.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoot(cmd.Context(), settings, args)
		},
	}
	return rootCommand
}

func runRoot(ctx context.Context, settings *types.Arguments, args []string) error {
	app := yutc.NewApp(ctx)
	return app.Run(ctx, args)
}
