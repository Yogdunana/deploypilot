package config

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"

	"github.com/Yogdunana/deploypilot/internal/tracing"
	"gopkg.in/natefinch/lumberjack.v2"
)

// InitLogger configures the global slog logger based on the provided LogConfig.
// If cfg.File is set, logs are written to both stdout and the specified file.
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

	// Build output writer: stdout + optional file
	var writers []io.Writer
	writers = append(writers, os.Stdout)

	if cfg.File != "" {
		// Parse max size (default 100MB)
		maxSize := parseMaxSize(cfg.MaxSize, 100)
		
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    maxSize, // MB
			MaxBackups: cfg.MaxBackups,
			MaxAge:     30, // days
			Compress:   true,
		}
		writers = append(writers, fileWriter)
	}

	multiWriter := io.MultiWriter(writers...)

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(multiWriter, handlerOpts)
	} else {
		handler = slog.NewTextHandler(multiWriter, handlerOpts)
	}

	handler = tracing.NewTraceHandler(handler)
	slog.SetDefault(slog.New(handler))
}

// parseMaxSize parses size string like "100MB" or "1GB" into MB.
func parseMaxSize(s string, defaultMB int) int {
	if s == "" {
		return defaultMB
	}
	
	// Try to parse as plain number (assumed MB)
	if mb, err := strconv.Atoi(s); err == nil {
		return mb
	}
	
	// Parse with suffix
	var num int
	var unit string
	if _, err := fmt.Sscanf(s, "%d%s", &num, &unit); err != nil {
		return defaultMB
	}
	
	switch unit {
	case "GB", "gb":
		return num * 1024
	case "MB", "mb":
		return num
	case "KB", "kb":
		return num / 1024
	default:
		return defaultMB
	}
}
