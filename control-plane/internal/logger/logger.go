// Package logger provides a thin wrapper around slog so all packages log the same way.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a slog.Logger configured for the given level string.
// Unknown levels fall back to info.
func New(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(h)
}
