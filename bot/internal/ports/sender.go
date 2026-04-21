package ports

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// MessageSender abstracts outbound Telegram messages so handlers stay decoupled from the Bot API client.
type MessageSender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
}
