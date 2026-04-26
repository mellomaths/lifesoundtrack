package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
)

const platformSource = "telegram"

// NewBot builds a *bot.Bot with save-album and /list handlers. The same instance can be used for daily sends (T019).
func NewBot(log *slog.Logger, token string, save *core.SaveService, lib *core.LibraryService) (*bot.Bot, error) {
	if save == nil {
		return nil, fmt.Errorf("nil save service")
	}
	if lib == nil {
		return nil, fmt.Errorf("nil library service")
	}
	opts := []bot.Option{
		bot.WithDefaultHandler(func(tctx context.Context, b *bot.Bot, u *models.Update) {
			if u == nil {
				return
			}
			if u.CallbackQuery != nil {
				handleCallback(tctx, log, b, u.CallbackQuery, save, lib)
				return
			}
			if u.Message != nil {
				handleMessage(tctx, log, b, u.Message, save, lib)
			}
		}),
	}
	return bot.New(token, opts...)
}

// Start begins long-polling until ctx is cancelled.
func Start(ctx context.Context, log *slog.Logger, tb *bot.Bot) error {
	if tb == nil {
		return fmt.Errorf("nil bot")
	}
	log.Info("telegram adapter running (long polling)")
	tb.Start(ctx)
	return nil
}

// Run starts long-polling. Private 1:1 text and callback queries for album disambig.
func Run(ctx context.Context, log *slog.Logger, token string, save *core.SaveService, lib *core.LibraryService) error {
	tb, err := NewBot(log, token, save, lib)
	if err != nil {
		return err
	}
	return Start(ctx, log, tb)
}

func handleCallback(ctx context.Context, log *slog.Logger, b *bot.Bot, q *models.CallbackQuery, save *core.SaveService, lib *core.LibraryService) {
	if q == nil || q.From.ID == 0 {
		return
	}
	chatID, ok := callbackChatID(q)
	if !ok {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: q.ID})
		return
	}
	if strings.HasPrefix(q.Data, "lpl:") {
		handleListCallback(ctx, log, b, q, chatID, lib)
		return
	}
	var n int
	switch {
	case q.Data == "aother":
		n = 3 // "Other" → refine-query path in core
	case strings.HasPrefix(q.Data, "apick:"):
		suffix := strings.TrimPrefix(q.Data, "apick:")
		var err error
		n, err = strconv.Atoi(suffix)
		if err != nil || n < 1 || n > 2 {
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: q.ID})
			return
		}
	default:
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: q.ID})
		return
	}
	ext, disp, u := userIdentity(&q.From)
	um, err := save.ProcessPickByIndex(ctx, platformSource, ext, disp, u, n)
	if err != nil {
		log.Error("callback pick", "err", err)
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: q.ID, Text: "Could not complete that.", ShowAlert: true})
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: internalErrCopy()})
		return
	}
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: q.ID})
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: um.Text})
}

func callbackChatID(q *models.CallbackQuery) (any, bool) {
	if q == nil {
		return 0, false
	}
	switch q.Message.Type {
	case models.MaybeInaccessibleMessageTypeMessage:
		if q.Message.Message != nil {
			return q.Message.Message.Chat.ID, true
		}
	case models.MaybeInaccessibleMessageTypeInaccessibleMessage:
		if q.Message.InaccessibleMessage != nil {
			return q.Message.InaccessibleMessage.Chat.ID, true
		}
	}
	return 0, false
}

func callbackMessageID(q *models.CallbackQuery) (int, bool) {
	if q == nil {
		return 0, false
	}
	switch q.Message.Type {
	case models.MaybeInaccessibleMessageTypeMessage:
		if q.Message.Message != nil {
			return q.Message.Message.ID, true
		}
	}
	return 0, false
}

func handleListCallback(ctx context.Context, log *slog.Logger, b *bot.Bot, q *models.CallbackQuery, chatID any, lib *core.LibraryService) {
	sessionID, page, ok := parseListCallbackData(q.Data)
	if !ok {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: q.ID})
		return
	}
	ext, _, _ := userIdentity(&q.From)
	lr, err := lib.HandleListPageJump(ctx, platformSource, ext, sessionID, page)
	if err != nil {
		log.Error("list callback", "err", err)
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: q.ID})
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: internalErrCopy()})
		return
	}
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: q.ID})
	msgID, hasMsg := callbackMessageID(q)
	var markup models.ReplyMarkup
	if lr.NeedsKeyboard {
		markup = listPaginationKeyboard(lr.SessionID, lr.CurrentPage, lr.TotalPages)
	}
	if hasMsg {
		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   msgID,
			Text:        lr.Text,
			ReplyMarkup: markup,
		})
		if err == nil {
			return
		}
		log.Warn("list edit message", "err", err)
	}
	params := &bot.SendMessageParams{ChatID: chatID, Text: lr.Text}
	if lr.NeedsKeyboard {
		params.ReplyMarkup = listPaginationKeyboard(lr.SessionID, lr.CurrentPage, lr.TotalPages)
	}
	_, _ = b.SendMessage(ctx, params)
}

// parseListCallbackData expects lpl:<uuid>:<1-based page>.
func parseListCallbackData(data string) (sessionID string, page int, ok bool) {
	if !strings.HasPrefix(data, "lpl:") {
		return "", 0, false
	}
	rest := strings.TrimPrefix(data, "lpl:")
	i := strings.LastIndex(rest, ":")
	if i <= 0 || i >= len(rest)-1 {
		return "", 0, false
	}
	sessionID = rest[:i]
	pageStr := rest[i+1:]
	if sessionID == "" {
		return "", 0, false
	}
	p, err := strconv.Atoi(pageStr)
	if err != nil || p < 1 {
		return "", 0, false
	}
	return sessionID, p, true
}

func listPaginationKeyboard(sessionID string, currentPage, totalPages int) *models.InlineKeyboardMarkup {
	if totalPages <= 1 || sessionID == "" {
		return nil
	}
	var row []models.InlineKeyboardButton
	if currentPage > 1 {
		row = append(row, models.InlineKeyboardButton{
			Text:         "Back",
			CallbackData: "lpl:" + sessionID + ":" + strconv.Itoa(currentPage-1),
		})
	}
	if currentPage < totalPages {
		row = append(row, models.InlineKeyboardButton{
			Text:         "Next",
			CallbackData: "lpl:" + sessionID + ":" + strconv.Itoa(currentPage+1),
		})
	}
	if len(row) == 0 {
		return nil
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{row}}
}

func handleMessage(ctx context.Context, log *slog.Logger, b *bot.Bot, msg *models.Message, save *core.SaveService, lib *core.LibraryService) {
	if msg == nil || msg.From == nil {
		return
	}
	if msg.Chat.Type != models.ChatTypePrivate {
		return
	}
	if msg.Text == "" {
		return
	}
	chatID := msg.Chat.ID
	text := msg.Text
	ext, disp, u := userIdentity(msg.From)

	if q, isAlbum := core.ParseAlbumLine(text); isAlbum {
		um, err := save.ProcessAlbumQuery(ctx, platformSource, ext, disp, u, q)
		if err != nil {
			log.Error("save album", "err", err)
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: internalErrCopy()})
			return
		}
		params := &bot.SendMessageParams{ChatID: chatID, Text: um.Text}
		if um.Outcome == core.OutcomeDisambig && len(um.AlbumButtonLabels) > 0 {
			params.ReplyMarkup = disambigKeyboard(um.AlbumButtonLabels)
		}
		_, _ = b.SendMessage(ctx, params)
		return
	}

	if kind, artistRest, isList := core.ParseListLine(text); isList {
		lr, err := lib.HandleList(ctx, platformSource, ext, kind, artistRest)
		if err != nil {
			log.Error("list", "err", err)
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: internalErrCopy()})
			return
		}
		params := &bot.SendMessageParams{ChatID: chatID, Text: lr.Text}
		if lr.Kind == core.ListReplyPage && lr.NeedsKeyboard {
			params.ReplyMarkup = listPaginationKeyboard(lr.SessionID, lr.CurrentPage, lr.TotalPages)
		}
		_, _ = b.SendMessage(ctx, params)
		return
	}

	if n, ok := core.OneBasedPickFromText(text); ok {
		um, err := save.ProcessPickByIndex(ctx, platformSource, ext, disp, u, n)
		if err != nil {
			log.Error("pick", "err", err)
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: internalErrCopy()})
			return
		}
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: um.Text})
		return
	}

	cmd := core.ParseTextMessage(text)
	reply := core.Reply(cmd)
	log.Info("private message", "domain_command", cmd.String(), "chat_id", chatID)
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: reply})
}

// telegramInlineButtonTextMax is Telegram's max label length for inline buttons.
const telegramInlineButtonTextMax = 64

func disambigKeyboard(albumLabels []string) *models.InlineKeyboardMarkup {
	// Core should pass at most 2 distinct labels; skip duplicate strings defensively
	// so we never show two identical button captions (SC-007).
	uniq := make([]string, 0, 2)
	seen := make(map[string]struct{}, len(albumLabels))
	for _, lab := range albumLabels {
		if len(uniq) >= 2 {
			break
		}
		if _, ok := seen[lab]; ok {
			continue
		}
		seen[lab] = struct{}{}
		uniq = append(uniq, lab)
	}
	rows := make([][]models.InlineKeyboardButton, 0, 3)
	for i, lab := range uniq {
		if i >= 2 {
			break
		}
		rows = append(rows, []models.InlineKeyboardButton{{
			Text:         truncateForButton(lab),
			CallbackData: "apick:" + strconv.Itoa(i+1),
		}})
	}
	rows = append(rows, []models.InlineKeyboardButton{{
		Text:         "Other",
		CallbackData: "aother",
	}})
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func truncateForButton(s string) string {
	if len(s) <= telegramInlineButtonTextMax {
		return s
	}
	runes := []rune(s)
	if len(runes) <= telegramInlineButtonTextMax {
		return s
	}
	return string(runes[:telegramInlineButtonTextMax-1]) + "…"
}

func userIdentity(u *models.User) (externalID, display, username string) {
	if u == nil {
		return "", "", ""
	}
	externalID = strconv.FormatInt(u.ID, 10)
	parts := make([]string, 0, 2)
	if strings.TrimSpace(u.FirstName) != "" {
		parts = append(parts, strings.TrimSpace(u.FirstName))
	}
	if strings.TrimSpace(u.LastName) != "" {
		parts = append(parts, strings.TrimSpace(u.LastName))
	}
	display = strings.Join(parts, " ")
	username = strings.TrimSpace(u.Username)
	return externalID, display, username
}

func internalErrCopy() string {
	return "Something went wrong. Please try again in a bit."
}
