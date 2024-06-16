package main

import "github.com/spf13/cobra"

func runForEachCommand(cmd *cobra.Command, args []string) error {
	YutcLog.Trace().Msg("yutc.runForEachCommand() called")
	err := parseCommon(cmd, args)
	if err != nil {
		return err
	}
	return nil
}
