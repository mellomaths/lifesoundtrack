package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"github.com/mellomaths/lifesoundtrack/bot/internal/app"
	"github.com/mellomaths/lifesoundtrack/bot/internal/configs"
	"github.com/mellomaths/lifesoundtrack/bot/internal/observability"
	"github.com/mellomaths/lifesoundtrack/bot/internal/recommendation"
	"github.com/mellomaths/lifesoundtrack/bot/internal/storage/postgres"
	"github.com/mellomaths/lifesoundtrack/bot/internal/telegram"
)

func main() {
	// os.Getenv does not read .env files; load optional env files for local dev.
	_ = godotenv.Load()
	_ = godotenv.Load("bot/.env")

	observability.InitTracing()
	logger := observability.NewLogger()

	cfg, err := configs.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		log.Fatalf("bot api: %v", err)
	}

	baseCtx := context.Background()
	pool, err := postgres.NewPool(baseCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	store := postgres.NewStore(pool)
	if err := store.Migrate(baseCtx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	sender := &telegram.BotAPISender{API: api}
	d := &app.Dispatcher{
		Sender:    sender,
		Interests: store,
		Log:       logger,
	}

	logger.InfoContext(context.Background(), "starting bot",
		slog.String("username", api.Self.UserName),
		slog.String("transport", string(cfg.Transport)))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.DailyRecommendationEnabled {
		go recommendation.RunDailyLoop(ctx, logger, sender, store, cfg.DailyRecommendationHourUTC)
	}

	if err := telegram.Run(ctx, logger, api, cfg, d); err != nil {
		log.Fatalf("run: %v", err)
	}
}
