package config

import (
	"context"
	"testing"

	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestLoadContext(t *testing.T) {
	ctx := context.Background()
	cmd := &cobra.Command{}
	settings := &types.Arguments{Output: "test"}
	tempDir := "/tmp/test"
	logger := zerolog.Nop()

	ctx, err := LoadContext(ctx, cmd, settings, tempDir, &logger)
	assert.NoError(t, err)

	assert.Equal(t, cmd, GetCommand(ctx))
	assert.Equal(t, settings, GetSettings(ctx))
	assert.Equal(t, tempDir, GetTempDir(ctx))
	assert.Equal(t, &logger, GetLogger(ctx))
	assert.NotNil(t, GetRunData(ctx))
}
