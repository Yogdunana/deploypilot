package config

import (
	"log/slog"
	"os"

	"github.com/Yogdunana/deploypilot/internal/tracing"
)

// InitLogger configures the global slog logger based on the provided LogConfig.
func InitLogger(cfg LogConfig) {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handlerOpts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	}

	handler = tracing.NewTraceHandler(handler)
	slog.SetDefault(slog.New(handler))
}
