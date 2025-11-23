// Package logging provides structured logging utilities using zerolog.
package logging

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
)

// InitLogger initializes and configures a zerolog Logger.
// It reads the log level from the levelOverride parameter or YUTC_LOG_LEVEL environment variable.
// If neither is set, it defaults to INFO level.
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
