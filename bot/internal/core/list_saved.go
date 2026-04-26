package core

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mellomaths/lifesoundtrack/bot/internal/store"
)

// ListReplyKind is the high-level /list outcome for adapters.
type ListReplyKind int

const (
	ListReplyEmptyLibrary ListReplyKind = iota
	ListReplyNoMatches
	ListReplyPage
	ListReplyNeedSessionHint
)

// ListReply is user-visible list output plus paging metadata for Telegram.
type ListReply struct {
	Kind          ListReplyKind
	Text          string
	SessionID     string
	CurrentPage   int
	TotalPages    int
	NeedsKeyboard bool
}

// LibraryService implements /list using the Postgres store.
type LibraryService struct {
	Store *store.Store
}

// HandleList parses list commands into a user-visible reply.
func (lib *LibraryService) HandleList(ctx context.Context, source, externalID string, kind ListParseKind, artistRaw string) (ListReply, error) {
	if lib == nil || lib.Store == nil {
		return ListReply{}, fmt.Errorf("nil library service")
	}
	listenerID, err := lib.Store.ListenerIDBySourceExternal(ctx, source, externalID)
	if err != nil {
		return ListReply{}, err
	}

	var artistNorm *string
	switch kind {
	case ListParseBareOrWhitespace, ListParseNext, ListParseBack:
		// no filter
	case ListParseArtistFilter:
		n := NormalizeArtistQuery(artistRaw)
		if n == "" {
			kind = ListParseBareOrWhitespace
		} else {
			artistNorm = &n
		}
	default:
		return ListReply{}, fmt.Errorf("unsupported list kind %v", kind)
	}

	switch kind {
	case ListParseNext:
		return lib.sessionStep(ctx, listenerID, +1)
	case ListParseBack:
		return lib.sessionStep(ctx, listenerID, -1)
	default:
		return lib.pageFirst(ctx, listenerID, artistNorm)
	}
}

func (lib *LibraryService) pageFirst(ctx context.Context, listenerID string, artistNorm *string) (ListReply, error) {
	count, err := lib.Store.CountSavedAlbumsForListener(ctx, listenerID, artistNorm)
	if err != nil {
		return ListReply{}, err
	}
	if count == 0 {
		if artistNorm == nil {
			return ListReply{Kind: ListReplyEmptyLibrary, Text: listEmptyLibraryCopy()}, nil
		}
		return ListReply{Kind: ListReplyNoMatches, Text: listNoMatchCopy()}, nil
	}
	totalPages := int((count + int64(ListPageSize) - 1) / int64(ListPageSize))
	rows, err := lib.Store.ListSavedAlbumsPage(ctx, listenerID, artistNorm, 0, ListPageSize)
	if err != nil {
		return ListReply{}, err
	}
	text := buildListText(rows, 1, totalPages)
	reply := ListReply{Kind: ListReplyPage, Text: text, CurrentPage: 1, TotalPages: totalPages}
	if totalPages > 1 {
		sid, err := lib.Store.InsertAlbumListSession(ctx, listenerID, artistNorm, 1, store.DefaultSessionTTL)
		if err != nil {
			return ListReply{}, err
		}
		reply.SessionID = sid
		reply.NeedsKeyboard = true
	}
	return reply, nil
}

func (lib *LibraryService) sessionStep(ctx context.Context, listenerID string, delta int) (ListReply, error) {
	sess, err := lib.Store.LatestOpenAlbumListSession(ctx, listenerID)
	if err != nil {
		return ListReply{}, err
	}
	if sess == nil {
		return ListReply{Kind: ListReplyNeedSessionHint, Text: listNeedSessionCopy()}, nil
	}
	count, err := lib.Store.CountSavedAlbumsForListener(ctx, listenerID, sess.ArtistFilterNorm)
	if err != nil {
		return ListReply{}, err
	}
	if count == 0 {
		return ListReply{Kind: ListReplyNeedSessionHint, Text: listNeedSessionCopy()}, nil
	}
	totalPages := int((count + int64(ListPageSize) - 1) / int64(ListPageSize))
	if totalPages <= 1 {
		return ListReply{Kind: ListReplyNeedSessionHint, Text: listNeedSessionCopy()}, nil
	}
	newPage := sess.CurrentPage + delta
	if newPage < 1 {
		newPage = 1
	}
	if newPage > totalPages {
		newPage = totalPages
	}
	offset := (newPage - 1) * ListPageSize
	rows, err := lib.Store.ListSavedAlbumsPage(ctx, listenerID, sess.ArtistFilterNorm, offset, ListPageSize)
	if err != nil {
		return ListReply{}, err
	}
	if err := lib.Store.UpdateAlbumListSessionPage(ctx, sess.ID, newPage); err != nil {
		return ListReply{}, err
	}
	text := buildListText(rows, newPage, totalPages)
	return ListReply{
		Kind:          ListReplyPage,
		Text:          text,
		SessionID:     sess.ID,
		CurrentPage:   newPage,
		TotalPages:    totalPages,
		NeedsKeyboard: true,
	}, nil
}

// HandleListPageJump loads session, validates listener + expiry + page bounds, returns a page (callback path).
func (lib *LibraryService) HandleListPageJump(ctx context.Context, source, externalID, sessionID string, page int) (ListReply, error) {
	if lib == nil || lib.Store == nil {
		return ListReply{}, fmt.Errorf("nil library service")
	}
	listenerID, err := lib.Store.ListenerIDBySourceExternal(ctx, source, externalID)
	if err != nil {
		return ListReply{}, err
	}
	if listenerID == "" {
		return ListReply{Kind: ListReplyNeedSessionHint, Text: listNeedSessionCopy()}, nil
	}
	sess, err := lib.Store.GetAlbumListSession(ctx, sessionID)
	if err != nil {
		return ListReply{}, err
	}
	if sess == nil {
		return ListReply{Kind: ListReplyNeedSessionHint, Text: listNeedSessionCopy()}, nil
	}
	if sess.ListenerID != listenerID {
		return ListReply{Kind: ListReplyNeedSessionHint, Text: listNeedSessionCopy()}, nil
	}
	if time.Now().After(sess.ExpiresAt) {
		return ListReply{Kind: ListReplyNeedSessionHint, Text: listNeedSessionCopy()}, nil
	}
	count, err := lib.Store.CountSavedAlbumsForListener(ctx, listenerID, sess.ArtistFilterNorm)
	if err != nil {
		return ListReply{}, err
	}
	totalPages := int((count + int64(ListPageSize) - 1) / int64(ListPageSize))
	if totalPages <= 1 || page < 1 || page > totalPages {
		return ListReply{Kind: ListReplyNeedSessionHint, Text: listNeedSessionCopy()}, nil
	}
	offset := (page - 1) * ListPageSize
	rows, err := lib.Store.ListSavedAlbumsPage(ctx, listenerID, sess.ArtistFilterNorm, offset, ListPageSize)
	if err != nil {
		return ListReply{}, err
	}
	if err := lib.Store.UpdateAlbumListSessionPage(ctx, sess.ID, page); err != nil {
		return ListReply{}, err
	}
	text := buildListText(rows, page, totalPages)
	return ListReply{
		Kind:          ListReplyPage,
		Text:          text,
		SessionID:     sess.ID,
		CurrentPage:   page,
		TotalPages:    totalPages,
		NeedsKeyboard: true,
	}, nil
}

const listLineMaxRunes = 240

func buildListText(rows []store.SavedAlbumListRow, page, totalPages int) string {
	var b strings.Builder
	for _, r := range rows {
		line := FormatSavedAlbumLine(r.Title, r.PrimaryArtist, r.Year)
		b.WriteString(truncateLineForList(line, listLineMaxRunes))
		b.WriteByte('\n')
	}
	if totalPages > 1 {
		b.WriteString(fmt.Sprintf("Page %d of %d\n", page, totalPages))
		b.WriteString("Use the buttons, or send /list next or /list back.")
	}
	return strings.TrimRight(b.String(), "\n")
}

func truncateLineForList(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return s
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-1]) + "…"
}

func listEmptyLibraryCopy() string {
	return "You have not saved any albums yet. Save one with /album followed by a title, artist, or a Spotify album link — for example: /album Kind of Blue"
}

func listNoMatchCopy() string {
	return "No saved albums match that artist. Try different spelling, or send /list to see everything you have saved."
}

func listNeedSessionCopy() string {
	return "That list has expired. Send /list again (add an artist after /list if you were filtering)."
}
