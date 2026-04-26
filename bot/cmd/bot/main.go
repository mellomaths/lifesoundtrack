package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/mellomaths/lifesoundtrack/bot/internal/adapter/telegram"
	"github.com/mellomaths/lifesoundtrack/bot/internal/config"
	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
	"github.com/mellomaths/lifesoundtrack/bot/internal/metadata"
	"github.com/mellomaths/lifesoundtrack/bot/internal/store"
	"github.com/robfig/cron/v3"
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
	lib := &core.LibraryService{Store: st}

	tb, err := telegram.NewBot(log, cfg.TelegramBotToken, save, lib)
	if err != nil {
		log.Error("telegram bot init", "err", err)
		os.Exit(1)
	}
	dailyMessenger := telegram.NewDailyMessenger(tb, log)
	dailyRunner := &core.DailyRecommendRunner{Store: st, Messenger: dailyMessenger, Log: log}

	log.Info("daily_recommendations_config",
		"enabled", cfg.DailyRecommendationsEnable,
		"tz", cfg.DailyRecommendationsTZName,
		"cron", cfg.DailyRecommendationsCron,
	)

	if cfg.DailyRecommendationsEnable && cfg.DailyRecommendationsLocation != nil {
		sched := cron.New(cron.WithLocation(cfg.DailyRecommendationsLocation))
		_, err := sched.AddFunc(cfg.DailyRecommendationsCron, func() {
			runID := uuid.New().String()
			log.Info("daily_recommendations_cron_tick", "run_id", runID)
			listCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			targets, lerr := st.ListTelegramDailyTargets(listCtx)
			if lerr != nil {
				log.Error("daily_recommendations_listeners", "run_id", runID, "err", lerr)
				return
			}
			log.Info("daily_recommendations_listeners", "run_id", runID, "eligible_count", len(targets))
			var skippedChat int
			for _, t := range targets {
				chatID, perr := strconv.ParseInt(t.ExternalID, 10, 64)
				if perr != nil {
					skippedChat++
					log.Warn("daily_recommendations_skip", "run_id", runID, "listener_id", t.ListenerID, "reason", "invalid_telegram_external_id")
					continue
				}
				dailyRunner.RunForListener(listCtx, runID, t.ListenerID, chatID)
			}
			if skippedChat > 0 {
				log.Info("daily_recommendations_tick_summary", "run_id", runID, "skipped_invalid_chat_id", skippedChat)
			}
		})
		if err != nil {
			log.Error("daily_recommendations_cron_register", "err", err)
			os.Exit(1)
		}
		go func() {
			sched.Start()
			<-ctx.Done()
			stopDone := sched.Stop()
			<-stopDone.Done()
		}()
	}

	if err := telegram.Start(ctx, log, tb); err != nil {
		log.Error("bot stopped with error", "err", err)
		os.Exit(1)
	}
}
