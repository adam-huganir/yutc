package main

import (
	"github.com/spf13/cobra"
)

func runForEachCommand(cmd *cobra.Command, args []string) error {
	//YutcLog.Trace().Msg("yutc.runForEachCommand() called")
	//templateFiles, dataFiles := parseInputs(args)
	//err := parseArgs(templateFiles, dataFiles)
	//if len(dataFiles) == 1 {
	//	// we assume that if the datafile is a list, we will iterate over it
	//	data, dataType, err := internal.GatherData(dataFiles, runSettings.Append)
	//	if err != nil {
	//		return err
	//	}
	//	fmt.Println(data, dataType)
	//} else {
	//	// we assume that each datafile is a separate iteration, and in this case we don't care what each individual
	//	// file contains
	//	for _, dataFile := range dataFiles {
	//		data, dataType, err := internal.GatherData([]string{dataFile}, runSettings.Append)
	//		if err != nil {
	//			return err
	//		}
	//		fmt.Println(data, dataType)
	//	}
	//}
	//if err != nil {
	//	return err
	//}
	return nil
}
