package main

import (
	"github.com/adam-huganir/yutc/internal"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"testing"
)

func newCmdTest(settings *internal.YutcSettings, args []string) *cobra.Command {
	cmd := newRootCommand()
	runSettings = settings
	initRoot(cmd, settings)
	cmd.SetArgs(args)
	return cmd
}

func Test_runRoot(t *testing.T) {
	cmd := newCmdTest(&internal.YutcSettings{}, []string{"--version"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Error executing command: %v", err)
	}
	assert.Equal(t, 0, exitCode)
}
