package observability

import (
	"log/slog"
	"os"
)

// NewLogger returns a JSON slog handler suitable for centralized log ingestion.
func NewLogger() *slog.Logger {
	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(h)
}
