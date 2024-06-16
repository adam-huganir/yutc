package main

import "github.com/spf13/cobra"

func runTemplateCommand(cmd *cobra.Command, args []string) error {
	YutcLog.Trace().Msg("yutc.runTemplateCommand() called")
	err := parseCommon(cmd, args)
	if err != nil {
		return err
	}
	return nil
}
