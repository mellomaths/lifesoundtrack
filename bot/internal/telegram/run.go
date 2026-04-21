package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mellomaths/lifesoundtrack/bot/internal/app"
	"github.com/mellomaths/lifesoundtrack/bot/internal/configs"
)

const telegramSecretHeader = "X-Telegram-Bot-Api-Secret-Token"

// Run receives updates until ctx is cancelled (polling or webhook per cfg).
func Run(ctx context.Context, log *slog.Logger, api *tgbotapi.BotAPI, cfg *configs.Config, d *app.Dispatcher) error {
	switch cfg.Transport {
	case configs.TransportPolling:
		return runPolling(ctx, log, api, d)
	case configs.TransportWebhook:
		return runWebhook(ctx, log, api, cfg, d)
	default:
		return nil
	}
}

func runPolling(ctx context.Context, log *slog.Logger, api *tgbotapi.BotAPI, d *app.Dispatcher) error {
	if _, err := api.Request(tgbotapi.DeleteWebhookConfig{}); err != nil {
		log.WarnContext(ctx, "deleteWebhook", slog.Any("err", err))
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := api.GetUpdatesChan(u)

	go func() {
		<-ctx.Done()
		api.StopReceivingUpdates()
	}()

	log.InfoContext(ctx, "transport configured", slog.String("transport", "polling"))

	for update := range updates {
		d.Dispatch(ctx, update)
	}
	return nil
}

func runWebhook(ctx context.Context, log *slog.Logger, api *tgbotapi.BotAPI, cfg *configs.Config, d *app.Dispatcher) error {
	fullURL, err := cfg.WebhookFullURL()
	if err != nil {
		return fmt.Errorf("webhook url: %w", err)
	}

	updates := make(chan tgbotapi.Update, api.Buffer)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc(cfg.WebhookPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cfg.WebhookSecretToken != "" {
			if got := r.Header.Get(telegramSecretHeader); got != cfg.WebhookSecretToken {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		upd, err := api.HandleUpdate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		select {
		case updates <- *upd:
		default:
			log.WarnContext(r.Context(), "updates channel full; dropping update",
				slog.Int("update_id", upd.UpdateID))
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	ln, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.ListenAddr, err)
	}

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.ErrorContext(ctx, "http server", slog.Any("err", err))
		}
	}()

	log.InfoContext(ctx, "transport configured",
		slog.String("transport", "webhook"),
		slog.String("listen_addr", cfg.ListenAddr),
		slog.String("path", cfg.WebhookPath),
		slog.String("public_url", fullURL))

	if err := setWebhook(api, fullURL, cfg.WebhookSecretToken); err != nil {
		_ = srv.Close()
		return fmt.Errorf("setWebhook: %w", err)
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.WarnContext(ctx, "http shutdown", slog.Any("err", err))
		}
		close(updates)
	}()

	for update := range updates {
		d.Dispatch(ctx, update)
	}
	return nil
}

func setWebhook(api *tgbotapi.BotAPI, webhookURL, secretToken string) error {
	params := tgbotapi.Params{}
	params["url"] = webhookURL
	params.AddNonEmpty("secret_token", secretToken)
	if _, err := api.MakeRequest("setWebhook", params); err != nil {
		return err
	}
	return nil
}
