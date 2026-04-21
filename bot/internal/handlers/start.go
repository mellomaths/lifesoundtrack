package handlers

import (
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// StartReplyText is the canonical body for /start replies (spec: commands/start.md).
func StartReplyText() string {
	return "Welcome to LifeSoundtrack — the soundtrack for your life. Use /start anytime to see this message again."
}

// HandleStart builds the reply for /start or returns an error when no reply can be sent.
func HandleStart(update tgbotapi.Update) (tgbotapi.Chattable, error) {
	if update.Message == nil {
		return nil, errors.New("no message in update")
	}
	if update.Message.Chat == nil || update.Message.Chat.ID == 0 {
		return nil, errors.New("missing chat")
	}
	cmd := update.Message.Command()
	if cmd != "start" {
		return nil, fmt.Errorf("not a start command: %q", cmd)
	}
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, StartReplyText())
	return msg, nil
}
