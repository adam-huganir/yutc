package main

import "github.com/spf13/cobra"

func runTemplateCommand(cmd *cobra.Command, args []string) error {
	YutcLog.Trace().Msg("yutc.runTemplateCommand() called")
	templateFiles, dataFiles := parseInputs(args)
	err := parseCommon(templateFiles, dataFiles)
	if err != nil {
		return err
	}
	return nil
}
