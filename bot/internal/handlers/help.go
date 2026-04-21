package handlers

import (
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HelpText is the canonical body for /help (spec: commands/help.md).
func HelpText() string {
	return `LifeSoundtrack — commands:
/start — Welcome message
/help — Show this help
/album <album title> - <artist> — Save an album you want to listen to (use " - " between title and artist)

Example: /album Abbey Road - The Beatles`
}

// HandleHelp builds the reply for /help.
func HandleHelp(update tgbotapi.Update) (tgbotapi.Chattable, error) {
	if update.Message == nil {
		return nil, errors.New("no message in update")
	}
	if update.Message.Chat == nil || update.Message.Chat.ID == 0 {
		return nil, errors.New("missing chat")
	}
	cmd := update.Message.Command()
	if cmd != "help" {
		return nil, fmt.Errorf("not a help command: %q", cmd)
	}
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, HelpText())
	return msg, nil
}
