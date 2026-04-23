package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Config holds process and adapter settings loaded from the environment.
type Config struct {
	TelegramBotToken string
	LogLevel         slog.Level
}

// FromEnv returns config for the first shipping adapter: Telegram. Names align with
// [../.env.example] and [../quickstart.md].
func FromEnv() (Config, error) {
	tok := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if tok == "" {
		return Config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	lvl := parseLogLevel(os.Getenv("LOG_LEVEL"))
	return Config{TelegramBotToken: tok, LogLevel: lvl}, nil
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERR", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
