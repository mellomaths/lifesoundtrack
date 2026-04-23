package telegram

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
)

// Run starts long-polling the Telegram API and dispatches private 1:1 text messages
// into [core] for replies. Non-private and non-text updates are ignored.
func Run(ctx context.Context, log *slog.Logger, token string) error {
	opts := []bot.Option{
		bot.WithDefaultHandler(func(tctx context.Context, b *bot.Bot, u *models.Update) {
			handleUpdate(tctx, log, b, u)
		}),
	}
	tb, err := bot.New(token, opts...)
	if err != nil {
		return fmt.Errorf("telegram bot init: %w", err)
	}
	log.Info("telegram adapter running (long polling)")
	tb.Start(ctx)
	return nil
}

func handleUpdate(ctx context.Context, log *slog.Logger, b *bot.Bot, u *models.Update) {
	if u == nil || u.Message == nil {
		return
	}
	msg := u.Message
	if msg.Chat.Type != models.ChatTypePrivate {
		return
	}
	if msg.Text == "" {
		// e.g. sticker/photo: no product requirement to reply; ignore.
		return
	}

	cmd := core.ParseTextMessage(msg.Text)
	reply := core.Reply(cmd)
	// No message body, no token: domain command and chat id for operator traces only.
	log.Info("private message", "domain_command", cmd.String(), "chat_id", msg.Chat.ID)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   reply,
	})
	if err != nil {
		log.Error("send message", "err", err, "domain_command", cmd.String(), "chat_id", msg.Chat.ID)
	}
}
