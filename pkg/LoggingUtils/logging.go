package LoggingUtils

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

const (
	LogLevelTrace slog.Level = -8
	LogLevelFatal slog.Level = 12
)

var (
	logLevel = strings.ToUpper(os.Getenv("YUTC_LOG_LEVEL"))
	LogType  = strings.ToUpper(os.Getenv("YUTC_LOG_TYPE"))
)

func NewLogger(h slog.Handler) *YutcLogger {
	if h == nil {
		panic("nil Handler")
	}
	return &YutcLogger{handler: h, Logger: slog.New(h)}
}

type YutcLogger struct {
	handler slog.Handler
	*slog.Logger
}

func (l *YutcLogger) Trace(msg string, args ...any) {
	l.Log(context.Background(), LogLevelTrace, msg, args...)
}

func (l *YutcLogger) Fatal(msg string, args ...any) {
	l.Log(context.Background(), LogLevelFatal, msg, args...)
}

func GetLogHandler() *YutcLogger {
	options := &slog.HandlerOptions{
		Level: GetLogLevel(),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "level" {
				if a.Value.Any().(slog.Level) == LogLevelTrace {
					a.Value = slog.StringValue("TRACE")
				} else if a.Value.Any().(slog.Level) == LogLevelFatal {
					a.Value = slog.StringValue("FATAL")
				}
			} else if a.Key == "time" {
			}
			return a
		},
	}
	var handler slog.Handler
	switch LogType {
	case "JSON":
		handler = slog.NewJSONHandler(os.Stderr, options)
	default:
		handler = slog.NewTextHandler(os.Stderr, options)
	}
	logger := NewLogger(handler)
	return logger
}

func GetLogLevel() slog.Level {
	switch logLevel {
	case "TRACE":
		return LogLevelTrace // -8
	case "DEBUG":
		return slog.LevelDebug // -4
	case "INFO":
		return slog.LevelInfo // 0
	case "WARN":
		return slog.LevelWarn // 4
	case "ERROR":
		return slog.LevelError // 8
	case "FATAL":
		return LogLevelFatal // 12
	default:
		return slog.LevelInfo
	}
}
