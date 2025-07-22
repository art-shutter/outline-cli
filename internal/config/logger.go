package config

import (
	"fmt"
	"log/slog"
	"os"
)

func levelFromString(logLevel string) slog.Level {
	switch logLevel {
	case "error":
		return slog.LevelError
	case "warning":
		return slog.LevelWarn
	case "":
		fallthrough
	case "info":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	default:
		panic(fmt.Sprintf("illegal log level (%s), you should not see this error", logLevel))
	}
}

func InitLogger(logLevel string) {
	level := levelFromString(logLevel)

	opts := &slog.HandlerOptions{
		Level: level,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)
}
