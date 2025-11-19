package main

import (
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
)

func newRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "yutc",
		Short: "Yet Unnamed Template CLI",
		Args:  cobra.MinimumNArgs(0),
		RunE:  runRoot,
	}
}

func runRoot(cmd *cobra.Command, args []string) (err error) {
	app := internal.NewApp(runSettings)
	return app.Run(args)
}
