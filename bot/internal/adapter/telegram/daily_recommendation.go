package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
)

// DailyMessenger sends daily album picks using an existing long-poll *bot.Bot (FR-017 / T019).
type DailyMessenger struct {
	b   *bot.Bot
	log *slog.Logger
}

// NewDailyMessenger wraps a Telegram bot client for proactive daily sends.
func NewDailyMessenger(b *bot.Bot, log *slog.Logger) *DailyMessenger {
	return &DailyMessenger{b: b, log: log}
}

// SendDailyPick implements core.DailyMessenger (photo + caption and/or text, optional URL button).
func (m *DailyMessenger) SendDailyPick(ctx context.Context, chatID int64, msg core.DailyPickMessage) error {
	if m == nil || m.b == nil {
		return fmt.Errorf("nil telegram daily messenger")
	}
	var markup models.ReplyMarkup
	if msg.UseURLButton && msg.SpotifyURL != "" {
		markup = spotifyURLKeyboardMarkup(msg.SpotifyURL)
	}
	if msg.ArtHTTPSURL != "" {
		err := m.sendPhotoOnce(ctx, chatID, msg, markup)
		if err != nil {
			if retry, d := telegramRetryAfter(err); retry {
				if m.log != nil {
					m.log.Warn("telegram_daily_429", "retry_after_sec", d/time.Second)
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(d):
				}
				err = m.sendPhotoOnce(ctx, chatID, msg, markup)
			}
		}
		return err
	}
	err := m.sendMessageOnce(ctx, chatID, msg, markup)
	if err != nil {
		if retry, d := telegramRetryAfter(err); retry {
			if m.log != nil {
				m.log.Warn("telegram_daily_429", "retry_after_sec", d/time.Second)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d):
			}
			err = m.sendMessageOnce(ctx, chatID, msg, markup)
		}
	}
	return err
}

func (m *DailyMessenger) sendPhotoOnce(ctx context.Context, chatID int64, msg core.DailyPickMessage, markup models.ReplyMarkup) error {
	_, err := m.b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:      chatID,
		Photo:       &models.InputFileString{Data: msg.ArtHTTPSURL},
		Caption:     truncateCaption(msg.BodyForText),
		ReplyMarkup: markup,
	})
	return err
}

func (m *DailyMessenger) sendMessageOnce(ctx context.Context, chatID int64, msg core.DailyPickMessage, markup models.ReplyMarkup) error {
	_, err := m.b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        msg.BodyForText,
		ReplyMarkup: markup,
	})
	return err
}

func spotifyURLKeyboardMarkup(url string) models.ReplyMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "Open in Spotify", URL: url}},
		},
	}
}

// Telegram caption limit is 1024 bytes for photos.
const maxPhotoCaptionRunes = 1000

func truncateCaption(s string) string {
	r := []rune(s)
	if len(r) <= maxPhotoCaptionRunes {
		return s
	}
	return string(r[:maxPhotoCaptionRunes-1]) + "…"
}

func telegramRetryAfter(err error) (bool, time.Duration) {
	if err == nil {
		return false, 0
	}
	msg := err.Error()
	if !strings.Contains(msg, "429") && !strings.Contains(strings.ToLower(msg), "too many requests") {
		return false, 0
	}
	// Best-effort: Retry-After may appear in error string from library.
	if i := strings.Index(msg, "retry after"); i >= 0 {
		// e.g. "retry after 12"
		var sec int
		_, scanErr := fmt.Sscanf(msg[i:], "retry after %d", &sec)
		if scanErr == nil && sec > 0 {
			return true, time.Duration(sec) * time.Second
		}
	}
	return true, 3 * time.Second
}
