package main

import (
	"fmt"
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
)

func runForEachCommand(cmd *cobra.Command, args []string) error {
	YutcLog.Trace().Msg("yutc.runForEachCommand() called")
	templateFiles, dataFiles := parseInputs(args)
	err := parseCommon(templateFiles, dataFiles)
	if len(dataFiles) == 1 {
		// we assume that if the datafile is a list, we will iterate over it
		data, dataType, err := internal.CollateData(dataFiles, false)
		if err != nil {
			return err
		}
		fmt.Println(data, dataType)
	} else {
		// we assume that each datafile is a separate iteration
	}
	if err != nil {
		return err
	}
	return nil
}
