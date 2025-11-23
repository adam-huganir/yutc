package main

import (
	"context"
	"testing"

	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/stretchr/testify/assert"
)

func Test_runRoot(t *testing.T) {
	settings := &types.Arguments{}
	runData := &types.RunData{}
	cmd := newRootCommand(settings, runData, &logger)
	ctx := context.Background()
	// ctx, _ = config.LoadContext(ctx, cmd, settings, "", &logger) // Removed
	initRoot(cmd, settings)
	cmd.SetArgs([]string{"--version"})
	err := cmd.ExecuteContext(ctx)
	assert.NoError(t, err)
}
