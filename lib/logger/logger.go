package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

// Init initializes the global logger with structured logging
// Outputs to stdout, which systemd captures and forwards to journalctl
func Init() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	Log = slog.New(handler)
	slog.SetDefault(Log)
}

// InitWithLevel initializes the logger with a specific log level
func InitWithLevel(level slog.Level) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	Log = slog.New(handler)
	slog.SetDefault(Log)
}
