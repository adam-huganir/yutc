package logging

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
)

func InitLogger(levelOverride string) zerolog.Logger {
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	logger := zerolog.New(output).With().Timestamp().Logger()
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
		logger.Info().Msg("Invalid log level, defaulting to INFO")
		level = zerolog.InfoLevel
	}
	logger = logger.Level(level)
	logger.Debug().Msg("Logger initialized to " + strings.ToUpper(level.String()))
	return logger
}
