// Package app wires command routing and depends on interfaces, not concrete Telegram clients.
package app

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.opentelemetry.io/otel"

	"github.com/mellomaths/lifesoundtrack/bot/internal/handlers"
	"github.com/mellomaths/lifesoundtrack/bot/internal/observability"
	"github.com/mellomaths/lifesoundtrack/bot/internal/ports"
)

// Dispatcher routes updates to command handlers.
type Dispatcher struct {
	Sender    ports.MessageSender
	Interests ports.InterestWriter
	Log       *slog.Logger
}

// Dispatch routes a Telegram update to handlers and sends replies.
func (d *Dispatcher) Dispatch(ctx context.Context, update tgbotapi.Update) {
	if d.Sender == nil || d.Log == nil {
		return
	}

	ctx, span := otel.Tracer(observability.TracerName).Start(ctx, "dispatch")
	defer span.End()

	if update.Message == nil || !update.Message.IsCommand() {
		return
	}
	switch update.Message.Command() {
	case "start":
		reply, err := handlers.HandleStart(update)
		if err != nil {
			d.Log.WarnContext(ctx, "start handler", slog.Any("err", err))
			return
		}
		if _, err := d.Sender.Send(reply); err != nil {
			d.Log.WarnContext(ctx, "send reply", slog.Any("err", err))
		}
	case "help":
		reply, err := handlers.HandleHelp(update)
		if err != nil {
			d.Log.WarnContext(ctx, "help handler", slog.Any("err", err))
			return
		}
		if _, err := d.Sender.Send(reply); err != nil {
			d.Log.WarnContext(ctx, "send reply", slog.Any("err", err))
		}
	case "album":
		if d.Interests == nil {
			d.Log.WarnContext(ctx, "album skipped", slog.String("reason", "interest writer not configured"))
			return
		}
		reply, err := handlers.HandleAlbum(ctx, update, d.Interests)
		if err != nil {
			d.Log.WarnContext(ctx, "album handler", slog.Any("err", err))
			return
		}
		if _, err := d.Sender.Send(reply); err != nil {
			d.Log.WarnContext(ctx, "send reply", slog.Any("err", err))
		}
	default:
		// Unimplemented commands are ignored until specified under spec/commands/.
	}
}
