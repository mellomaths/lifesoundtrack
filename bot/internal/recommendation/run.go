// Package recommendation runs scheduled daily album picks and Telegram sends.
package recommendation

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mellomaths/lifesoundtrack/bot/internal/ports"
)

// RunDailyLoop sleeps until each scheduled UTC hour, then sends one fair-random recommendation per Telegram user who has albums.
func RunDailyLoop(ctx context.Context, log *slog.Logger, sender ports.MessageSender, store ports.RecommendationStore, hourUTC int) {
	log.InfoContext(ctx, "daily recommendation scheduler started", slog.Int("hour_utc", hourUTC))

	for {
		wait := DurationUntilNextHourUTC(hourUTC, time.Now().UTC())
		log.InfoContext(ctx, "daily recommendation waiting", slog.Duration("until_next_run", wait))

		select {
		case <-ctx.Done():
			log.InfoContext(ctx, "daily recommendation scheduler stopped")
			return
		case <-time.After(wait):
			runDailyBatch(ctx, log, sender, store)
		}
	}
}

// DurationUntilNextHourUTC returns time until the next occurrence of hourUTC (0–23) on the UTC clock.
func DurationUntilNextHourUTC(hourUTC int, nowUTC time.Time) time.Duration {
	if hourUTC < 0 {
		hourUTC = 0
	}
	if hourUTC > 23 {
		hourUTC = 23
	}
	next := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), hourUTC, 0, 0, 0, time.UTC)
	if !next.After(nowUTC) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(nowUTC)
}

func runDailyBatch(ctx context.Context, log *slog.Logger, sender ports.MessageSender, store ports.RecommendationStore) {
	recipients, err := store.ListTelegramRecipientsWithAlbums(ctx)
	if err != nil {
		log.WarnContext(ctx, "daily recommendation list recipients", slog.Any("err", err))
		return
	}

	for _, r := range recipients {
		pick, err := store.PickNextRecommendation(ctx, r.InternalUserID)
		if err != nil {
			log.WarnContext(ctx, "daily recommendation pick",
				slog.Int64("user_id", r.InternalUserID), slog.Any("err", err))
			continue
		}
		if pick == nil {
			continue
		}

		text := fmt.Sprintf(
			`Your LifeSoundtrack pick for today: "%s" by %s — enjoy the day.`,
			pick.AlbumTitle, pick.Artist,
		)
		msg := tgbotapi.NewMessage(r.TelegramChatID, text)
		if _, err := sender.Send(msg); err != nil {
			log.WarnContext(ctx, "daily recommendation send",
				slog.Int64("telegram_chat_id", r.TelegramChatID), slog.Any("err", err))
			continue
		}

		err = store.RecordRecommendation(ctx, ports.RecordRecommendationInput{
			AlbumInterestID: pick.AlbumInterestID,
			UserID:          pick.UserID,
			AlbumTitle:      pick.AlbumTitle,
			Artist:          pick.Artist,
		})
		if err != nil {
			log.WarnContext(ctx, "daily recommendation record",
				slog.Int64("user_id", pick.UserID), slog.Any("err", err))
			continue
		}

		log.InfoContext(ctx, "daily recommendation sent",
			slog.Int64("user_id", pick.UserID),
			slog.String("album", pick.AlbumTitle),
			slog.String("artist", pick.Artist))

		select {
		case <-ctx.Done():
			return
		case <-time.After(75 * time.Millisecond):
		}
	}
}
