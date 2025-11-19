package main

import (
	"testing"

	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/stretchr/testify/assert"
)

func Test_runRoot(t *testing.T) {
	settings := &types.YutcSettings{}
	cmd := newRootCommand(settings)
	initRoot(cmd, settings)
	cmd.SetArgs([]string{"--version"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Error executing command: %v", err)
	}
	assert.NoError(t, err)
}
