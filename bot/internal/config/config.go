package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Config holds process and adapter settings loaded from the environment.
type Config struct {
	TelegramBotToken     string
	LogLevel             slog.Level
	DatabaseURL          string
	MigrationsPath       string
	LastfmAPIKey         string
	MusicBrainzUserAgent string
	AutoMigrate          bool
}

// FromEnv returns config for the bot. DATABASE_URL and TELEGRAM_BOT_TOKEN are required
// for the 003 save-album and Telegram flows.
func FromEnv() (Config, error) {
	tok := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if tok == "" {
		return Config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	migrationsPath := strings.TrimSpace(os.Getenv("MIGRATIONS_PATH"))
	lvl := parseLogLevel(os.Getenv("LOG_LEVEL"))
	lastfm := strings.TrimSpace(os.Getenv("LASTFM_API_KEY"))
	ua := strings.TrimSpace(os.Getenv("MUSICBRAINZ_USER_AGENT"))
	if ua == "" {
		ua = "LifeSoundTrackBot/1.0 (https://github.com/mellomaths/lifesoundtrack)"
	}
	autoM := strings.EqualFold(strings.TrimSpace(os.Getenv("AUTO_MIGRATE")), "true") ||
		strings.TrimSpace(os.Getenv("AUTO_MIGRATE")) == "1"
	return Config{
		TelegramBotToken:     tok,
		LogLevel:             lvl,
		DatabaseURL:          databaseURL,
		MigrationsPath:       migrationsPath,
		LastfmAPIKey:         lastfm,
		MusicBrainzUserAgent: ua,
		AutoMigrate:          autoM,
	}, nil
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
