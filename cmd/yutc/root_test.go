package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/adam-huganir/yutc/pkg"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/stretchr/testify/assert"
)

func Test_runRoot(t *testing.T) {
	settings := &types.Arguments{}
	runData := &yutc.RunData{}
	cmd := newRootCommand(settings, runData, &logger)
	ctx := context.Background()
	initRoot(cmd, settings)
	cmd.SetArgs([]string{"--version"})
	err := cmd.ExecuteContext(ctx)
	assert.NoError(t, err)
}

func TestHelpTopic(t *testing.T) {
	settings := &types.Arguments{}
	runData := &yutc.RunData{}
	cmd := newRootCommand(settings, runData, &logger)
	initRoot(cmd, settings)
	ctx := context.Background()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help", "syntax"})

	err := cmd.ExecuteContext(ctx)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(buf.String(), "Argument syntax help"))
}
