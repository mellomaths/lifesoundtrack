package handlers

import (
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestHelpText_listsAlbumUsage(t *testing.T) {
	t.Parallel()
	got := HelpText()
	for _, needle := range []string{"/start", "/help", "/album", " - "} {
		if !strings.Contains(got, needle) {
			t.Fatalf("help text missing %q:\n%s", needle, got)
		}
	}
	if !strings.Contains(got, "LifeSoundtrack") {
		t.Fatal("help should mention LifeSoundtrack")
	}
}

func TestHandleHelp_ok(t *testing.T) {
	t.Parallel()
	upd := tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			Chat:      &tgbotapi.Chat{ID: 9, Type: "private"},
			Text:      "/help",
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: 5},
			},
		},
	}
	got, err := HandleHelp(upd)
	if err != nil {
		t.Fatal(err)
	}
	msg := got.(tgbotapi.MessageConfig)
	if msg.ChatID != 9 {
		t.Fatalf("chat id %d", msg.ChatID)
	}
	if !strings.Contains(msg.Text, "/album") {
		t.Fatalf("expected /album in reply: %q", msg.Text)
	}
}
