package main

import (
	"context"

	yutc "github.com/adam-huganir/yutc/pkg"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func newRootCommand(settings *types.Arguments, runData *yutc.RunData, logger *zerolog.Logger) *cobra.Command {
	rootCommand := &cobra.Command{
		Use:   "yutc [flags] <template_files...>",
		Short: "yutc - Yet Unnamed Templating CLI",
		Long:  `yutc is a command line tool for rendering complex templates from arbitrary sources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoot(cmd.Context(), settings, runData, logger, cmd, args)
		},
		SilenceUsage: true,
	}
	return rootCommand
}

func runRoot(ctx context.Context, settings *types.Arguments, runData *yutc.RunData, logger *zerolog.Logger, cmd *cobra.Command, args []string) error {
	app := yutc.NewApp(settings, runData, logger, cmd)
	return app.Run(ctx, args)
}
