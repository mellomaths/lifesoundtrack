package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mellomaths/lifesoundtrack/bot/internal/ports"
)

// BotAPISender adapts *tgbotapi.BotAPI to ports.MessageSender.
type BotAPISender struct {
	API *tgbotapi.BotAPI
}

// Send delegates to the underlying Bot API client.
func (s *BotAPISender) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	return s.API.Send(c)
}

var _ ports.MessageSender = (*BotAPISender)(nil)
