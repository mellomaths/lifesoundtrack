package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mellomaths/lifesoundtrack/bot/internal/ports"
)

const albumArgDelimiter = " - "

var errAlbumBadArgs = errors.New("album arguments invalid")

// HandleAlbum parses /album and persists via InterestWriter (spec: commands/album.md).
func HandleAlbum(ctx context.Context, update tgbotapi.Update, w ports.InterestWriter) (tgbotapi.Chattable, error) {
	if update.Message == nil {
		return nil, errors.New("no message in update")
	}
	if update.Message.Chat == nil || update.Message.Chat.ID == 0 {
		return nil, errors.New("missing chat")
	}
	if update.Message.From == nil {
		return userFacingChat(update.Message.Chat.ID, "I can't save an album without knowing your Telegram user. Try again from a normal user message."), nil
	}
	cmd := update.Message.Command()
	if cmd != "album" {
		return nil, fmt.Errorf("not an album command: %q", cmd)
	}

	args := strings.TrimSpace(update.Message.CommandArguments())
	title, artist, err := parseAlbumArgs(args)
	if err != nil {
		return userFacingChat(update.Message.Chat.ID, albumUsageHint()), nil
	}

	in := ports.AddInterestInput{
		Source:     ports.SourceTelegram,
		ExternalID: strconv.FormatInt(update.Message.From.ID, 10),
		AlbumTitle: title,
		Artist:     artist,
	}
	in.DisplayName = telegramDisplayName(update.Message.From)
	in.Username = telegramUsername(update.Message.From)

	if w == nil {
		return nil, errors.New("interest writer is nil")
	}
	if err := w.AddInterest(ctx, in); err != nil {
		return userFacingChat(update.Message.Chat.ID, "Something went wrong saving your album. Please try again later."), nil
	}

	body := fmt.Sprintf("Got it — saved %q by %s for your LifeSoundtrack list.", title, artist)
	return tgbotapi.NewMessage(update.Message.Chat.ID, body), nil
}

func parseAlbumArgs(args string) (title, artist string, err error) {
	if args == "" {
		return "", "", errAlbumBadArgs
	}
	i := strings.Index(args, albumArgDelimiter)
	if i < 0 {
		return "", "", errAlbumBadArgs
	}
	title = strings.TrimSpace(args[:i])
	artist = strings.TrimSpace(args[i+len(albumArgDelimiter):])
	if title == "" || artist == "" {
		return "", "", errAlbumBadArgs
	}
	return title, artist, nil
}

func telegramDisplayName(u *tgbotapi.User) *string {
	if u == nil {
		return nil
	}
	var parts []string
	if t := strings.TrimSpace(u.FirstName); t != "" {
		parts = append(parts, t)
	}
	if t := strings.TrimSpace(u.LastName); t != "" {
		parts = append(parts, t)
	}
	if len(parts) == 0 {
		return nil
	}
	s := strings.Join(parts, " ")
	return &s
}

func telegramUsername(u *tgbotapi.User) *string {
	if u == nil {
		return nil
	}
	s := strings.TrimPrefix(strings.TrimSpace(u.UserName), "@")
	if s == "" {
		return nil
	}
	return &s
}

func albumUsageHint() string {
	return `Usage: /album <album title> - <artist>

Put the album title first, then a space, hyphen, space, then the artist.

Example: /album Abbey Road - The Beatles`
}

func userFacingChat(chatID int64, text string) tgbotapi.Chattable {
	return tgbotapi.NewMessage(chatID, text)
}
