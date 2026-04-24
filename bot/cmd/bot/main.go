package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mellomaths/lifesoundtrack/bot/internal/adapter/telegram"
	"github.com/mellomaths/lifesoundtrack/bot/internal/config"
	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
	"github.com/mellomaths/lifesoundtrack/bot/internal/metadata"
	"github.com/mellomaths/lifesoundtrack/bot/internal/store"
)

func main() {
	if err := config.LoadLocalDotEnv(); err != nil {
		slog.Error("config bootstrap failed", "reason", "dotenv", "err", err)
		os.Exit(1)
	}
	cfg, err := config.FromEnv()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	migDir := cfg.MigrationsPath
	if migDir == "" {
		migDir = "migrations"
	}
	if !filepath.IsAbs(migDir) {
		abs, aerr := filepath.Abs(migDir)
		if aerr == nil {
			migDir = abs
		}
	}

	if cfg.AutoMigrate {
		if err := store.RunMigrations(migDir, cfg.DatabaseURL); err != nil {
			log.Error("migrations", "err", err)
			os.Exit(1)
		}
	}

	st, err := store.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	orch := metadata.NewChain(metadata.ChainConfig{
		LastfmAPIKey:         cfg.LastfmAPIKey,
		MusicBrainzUserAgent: cfg.MusicBrainzUserAgent,
		SpotifyClientID:      cfg.SpotifyClientID,
		SpotifyClientSecret:  cfg.SpotifyClientSecret,
		EnableSpotify:        cfg.MetadataEnableSpotify,
		EnableITunes:         cfg.MetadataEnableITunes,
		EnableLastfm:         cfg.MetadataEnableLastfm,
		EnableMusicBrainz:    cfg.MetadataEnableMusicBrainz,
		Log:                  log,
	})
	save := &core.SaveService{Store: st, Search: orch, Log: log}

	if err := telegram.Run(ctx, log, cfg.TelegramBotToken, save); err != nil {
		log.Error("bot stopped with error", "err", err)
		os.Exit(1)
	}
}
