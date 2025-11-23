package main

import (
	"context"
	"testing"

	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/stretchr/testify/assert"
)

func Test_runRoot(t *testing.T) {
	settings := &types.Arguments{}
	cmd := newRootCommand(settings)
	ctx := context.Background()
	initRoot(ctx, cmd, settings)
	cmd.SetArgs([]string{"--version"})
	err := cmd.ExecuteContext(ctx)
	assert.NoError(t, err)
}
