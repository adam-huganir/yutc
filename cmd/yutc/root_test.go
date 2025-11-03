package main

import (
	"testing"

	"github.com/adam-huganir/yutc/internal"
	"github.com/stretchr/testify/assert"
)

func Test_runRoot(t *testing.T) {
	cmd := newCmdTest(&internal.YutcSettings{}, []string{"--version"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Error executing command: %v", err)
	}
	assert.Equal(t, 0, *internal.ExitCode)
}
