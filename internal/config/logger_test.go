package config

import (
	"log/slog"
	"testing"
)

func TestInitLogger_InfoLevel(t *testing.T) {
	InitLogger(LogConfig{Level: "info", Format: "text"})
	slog.Info("test info message", "key", "value")
}

func TestInitLogger_DebugLevel(t *testing.T) {
	InitLogger(LogConfig{Level: "debug", Format: "json"})
	slog.Debug("test debug message")
}

func TestInitLogger_WarnLevel(t *testing.T) {
	InitLogger(LogConfig{Level: "warn", Format: "text"})
	slog.Warn("test warn message")
}

func TestInitLogger_ErrorLevel(t *testing.T) {
	InitLogger(LogConfig{Level: "error", Format: "json"})
	slog.Error("test error message")
}

func TestInitLogger_DefaultLevel(t *testing.T) {
	InitLogger(LogConfig{Level: "unknown-level", Format: "text"})
	// Unknown level should default to info
	slog.Info("test default level message")
}

func TestInitLogger_JSONFormat(t *testing.T) {
	InitLogger(LogConfig{Level: "info", Format: "json"})
	slog.Info("json format test", "key", "value")
}

func TestInitLogger_TextFormat(t *testing.T) {
	InitLogger(LogConfig{Level: "info", Format: "text"})
	slog.Info("text format test", "key", "value")
}

func TestInitLogger_EmptyConfig(t *testing.T) {
	InitLogger(LogConfig{})
	slog.Info("empty config test")
}
