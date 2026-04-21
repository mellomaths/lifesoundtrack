package handlers

import (
	"context"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mellomaths/lifesoundtrack/bot/internal/ports"
)

type captureInterest struct {
	last ports.AddInterestInput
	err  error
}

func (c *captureInterest) AddInterest(ctx context.Context, in ports.AddInterestInput) error {
	c.last = in
	return c.err
}

func TestHandleAlbum_success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cap := &captureInterest{}
	text := "/album Abbey Road - The Beatles"
	upd := tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
			From: &tgbotapi.User{
				ID:        100,
				FirstName: "Ada",
				LastName:  "Lovelace",
				UserName:  "adal",
			},
			Text: text,
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: 6},
			},
		},
	}
	got, err := HandleAlbum(ctx, upd, cap)
	if err != nil {
		t.Fatal(err)
	}
	msg := got.(tgbotapi.MessageConfig)
	if !strings.Contains(msg.Text, "Abbey Road") || !strings.Contains(msg.Text, "The Beatles") {
		t.Fatalf("unexpected reply: %q", msg.Text)
	}

	if cap.last.Source != ports.SourceTelegram || cap.last.ExternalID != "100" {
		t.Fatalf("identity: %+v", cap.last)
	}
	if cap.last.AlbumTitle != "Abbey Road" || cap.last.Artist != "The Beatles" {
		t.Fatalf("album parse: %+v", cap.last)
	}
	if cap.last.DisplayName == nil || *cap.last.DisplayName != "Ada Lovelace" {
		t.Fatalf("display name: %+v", cap.last.DisplayName)
	}
	if cap.last.Username == nil || *cap.last.Username != "adal" {
		t.Fatalf("username: %+v", cap.last.Username)
	}
}

func TestHandleAlbum_missingFrom(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	upd := tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			Chat:      &tgbotapi.Chat{ID: 1, Type: "channel"},
			Text:      "/album Foo - Bar",
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: 6},
			},
		},
	}
	got, err := HandleAlbum(ctx, upd, &captureInterest{})
	if err != nil {
		t.Fatal(err)
	}
	msg := got.(tgbotapi.MessageConfig)
	if !strings.Contains(msg.Text, "Telegram user") {
		t.Fatalf("reply: %q", msg.Text)
	}
}

func TestHandleAlbum_badArgs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	upd := tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			Chat:      &tgbotapi.Chat{ID: 2, Type: "private"},
			From:      &tgbotapi.User{ID: 1},
			Text:      "/album",
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: 6},
			},
		},
	}
	c := &captureInterest{}
	got, err := HandleAlbum(ctx, upd, c)
	if err != nil {
		t.Fatal(err)
	}
	if c.last.ExternalID != "" {
		t.Fatal("should not persist on bad args")
	}
	msg := got.(tgbotapi.MessageConfig)
	if !strings.Contains(msg.Text, "Usage") {
		t.Fatalf("expected usage hint: %q", msg.Text)
	}
}
