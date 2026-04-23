package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mellomaths/lifesoundtrack/bot/internal/adapter/telegram"
	"github.com/mellomaths/lifesoundtrack/bot/internal/config"
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

	if err := telegram.Run(ctx, log, cfg.TelegramBotToken); err != nil {
		log.Error("bot stopped with error", "err", err)
		os.Exit(1)
	}
}
