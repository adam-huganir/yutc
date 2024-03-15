package internal

import (
	"github.com/rs/zerolog"
	"os"
)

var YutcLog zerolog.Logger

func InitLogger() {
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	YutcLog = zerolog.New(output).With().Timestamp().Logger()
	loglevelString := os.Getenv("YUTC_LOG_LEVEL")
	level, err := zerolog.ParseLevel(loglevelString)
	if err != nil {
		YutcLog.Info().Msg("Invalid log level, defaulting to INFO")
		level = zerolog.InfoLevel
	}
	YutcLog = YutcLog.Level(level)
}
