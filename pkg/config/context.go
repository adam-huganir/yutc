package config

import (
	"context"

	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

type YutcContextKey string

const (
	AppKey      YutcContextKey = "app"
	TempDirKey  YutcContextKey = "tempDir"
	SettingsKey YutcContextKey = "settings"
	RunDataKey  YutcContextKey = "runData"
	LoggerKey   YutcContextKey = "logger"
	CommandKey  YutcContextKey = "command"
)

func LoadContext(ctx context.Context, cmd *cobra.Command, settings types.Arguments, tempDir string) (context.Context, error) {
	ctx = context.WithValue(ctx, CommandKey, cmd)
	ctx = context.WithValue(ctx, SettingsKey, &settings)
	ctx = context.WithValue(ctx, TempDirKey, tempDir)
	ctx = context.WithValue(ctx, RunDataKey, &types.RunData{})
	return ctx, nil
}

func GetSettings(ctx context.Context) *types.Arguments {
	return ctx.Value(SettingsKey).(*types.Arguments)
}

func GetRunData(ctx context.Context) *types.RunData {
	return ctx.Value(RunDataKey).(*types.RunData)
}

func GetTempDir(ctx context.Context) string {
	return ctx.Value(TempDirKey).(string)
}

func GetLogger(ctx context.Context) zerolog.Logger {
	return ctx.Value(LoggerKey).(zerolog.Logger)
}

func GetCommand(ctx context.Context) *cobra.Command {
	return ctx.Value(CommandKey).(*cobra.Command)
}
