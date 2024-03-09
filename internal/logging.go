package internal

import (
	"log/slog"
	"os"
	"strings"
)

var LogLevel = strings.ToUpper(os.Getenv("YUTC_LOG_LEVEL"))
var LogType = strings.ToUpper(os.Getenv("YUTC_LOG_TYPE"))

func GetLogHandler() *slog.Logger {
	options := &slog.HandlerOptions{Level: GetLogLevel()}
	var handler slog.Handler
	switch LogType {
	case "JSON":
		handler = slog.NewJSONHandler(os.Stderr, options)
	default:
		handler = slog.NewTextHandler(os.Stderr, options)
	}
	logger := slog.New(handler)
	return logger
}

func GetLogLevel() slog.Level {
	switch LogLevel {
	case "TRACE":
		return slog.Level(-8) // -8
	case "DEBUG":
		return slog.LevelDebug // -4
	case "INFO":
		return slog.LevelInfo // 0
	case "WARN":
		return slog.LevelWarn // 4
	case "ERROR":
		return slog.LevelError // 8
	case "FATAL":
		return slog.Level(12) // 12
	default:
		return slog.LevelInfo
	}
}
