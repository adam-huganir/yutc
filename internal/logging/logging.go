package logging

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
)

var YutcLog zerolog.Logger

func InitLogger(levelOverride string) {
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	YutcLog = zerolog.New(output).With().Timestamp().Logger()
	var loglevelString string
	if levelOverride != "" {
		loglevelString = levelOverride
	} else {
		loglevelString = os.Getenv("YUTC_LOG_LEVEL")
	}
	if loglevelString == "" {
		// default to info
		loglevelString = "info"
	}
	level, err := zerolog.ParseLevel(loglevelString)
	if err != nil || level == zerolog.NoLevel {
		YutcLog.Info().Msg("Invalid log level, defaulting to INFO")
		level = zerolog.InfoLevel
	}
	YutcLog = YutcLog.Level(level)
	YutcLog.Debug().Msg("Logger initialized to " + strings.ToUpper(level.String()))
}
