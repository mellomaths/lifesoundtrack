package handlers

import (
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Mirrors spec/commands/start.md Given/When/Then scenarios.
func TestHandleStart_privateStart(t *testing.T) {
	t.Parallel()
	upd := newCommandUpdate("/start", 123)
	got, err := HandleStart(upd)
	if err != nil {
		t.Fatal(err)
	}
	msg, ok := got.(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", got)
	}
	if msg.ChatID != 123 {
		t.Fatalf("chat id: got %d want 123", msg.ChatID)
	}
	if !strings.Contains(msg.Text, "LifeSoundtrack") {
		t.Fatalf("reply should mention LifeSoundtrack: %q", msg.Text)
	}
	if strings.TrimSpace(msg.Text) == "" {
		t.Fatal("reply must be non-empty")
	}
}

func TestHandleStart_privateStartWithAtSuffix(t *testing.T) {
	t.Parallel()
	text := "/start@OtherBot"
	upd := tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			Chat:      &tgbotapi.Chat{ID: 7, Type: "private"},
			Text:      text,
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: len([]rune(text))},
			},
		},
	}
	got, err := HandleStart(upd)
	if err != nil {
		t.Fatal(err)
	}
	msg := got.(tgbotapi.MessageConfig)
	if !strings.Contains(msg.Text, "LifeSoundtrack") {
		t.Fatalf("reply should mention LifeSoundtrack: %q", msg.Text)
	}
}

func TestHandleStart_noMessage(t *testing.T) {
	t.Parallel()
	_, err := HandleStart(tgbotapi.Update{})
	if err == nil {
		t.Fatal("expected error when update has no message")
	}
}

func TestHandleStart_missingChat(t *testing.T) {
	t.Parallel()
	upd := tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			Text:      "/start",
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: 6},
			},
		},
	}
	_, err := HandleStart(upd)
	if err == nil {
		t.Fatal("expected error when chat is missing")
	}
}

func newCommandUpdate(cmd string, chatID int64) tgbotapi.Update {
	return tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			Chat:      &tgbotapi.Chat{ID: chatID, Type: "private"},
			Text:      cmd,
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: len([]rune(cmd))},
			},
		},
	}
}
