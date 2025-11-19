package main

import (
	"testing"

	"github.com/adam-huganir/yutc/internal/config"
	"github.com/adam-huganir/yutc/internal/types"
	"github.com/stretchr/testify/assert"
)

func Test_runRoot(t *testing.T) {
	cmd := newCmdTest(&types.YutcSettings{}, []string{"--version"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Error executing command: %v", err)
	}
	assert.Equal(t, 0, *config.ExitCode)
}
