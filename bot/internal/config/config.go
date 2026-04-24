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
	SpotifyClientID      string
	SpotifyClientSecret  string
	// Metadata feature flags: unset env → true (opt-out). See [spec FR-002].
	MetadataEnableSpotify     bool
	MetadataEnableITunes      bool
	MetadataEnableLastfm      bool
	MetadataEnableMusicBrainz bool
	AutoMigrate               bool
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
		SpotifyClientID:      strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_ID")),
		SpotifyClientSecret:  strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_SECRET")),
		MetadataEnableSpotify:     parseMetadataFeatureFlag("LST_METADATA_ENABLE_SPOTIFY"),
		MetadataEnableITunes:      parseMetadataFeatureFlag("LST_METADATA_ENABLE_ITUNES"),
		MetadataEnableLastfm:      parseMetadataFeatureFlag("LST_METADATA_ENABLE_LASTFM"),
		MetadataEnableMusicBrainz: parseMetadataFeatureFlag("LST_METADATA_ENABLE_MUSICBRAINZ"),
		AutoMigrate:               autoM,
	}, nil
}

// parseMetadataFeatureFlag returns true by default. Only explicit false/0/no/off
// (case-insensitive) disable the flag; unset or empty is enabled.
func parseMetadataFeatureFlag(key string) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return true
	}
	if strings.TrimSpace(v) == "" {
		return true
	}
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "0", "false", "no", "off", "n", "f":
		return false
	default:
		return true
	}
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
