package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/mellomaths/lifesoundtrack/bot/internal/store"
)

const removeDisambigKind = "remove_saved"

// removeDisambigRoot is the JSON object stored in disambiguation_sessions for /remove multi-match.
type removeDisambigRoot struct {
	Kind       string               `json:"kind"`
	Candidates []removeDisambigItem `json:"candidates"`
}

type removeDisambigItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// RemoveReply is the result of [HandleRemove] for hosts that need disambiguation metadata (e.g. Telegram inline keyboards).
// When [DisambigSessionID] is non-empty, [ButtonLabels] has one entry per choice in order (1..N).
type RemoveReply struct {
	Text              string
	DisambigSessionID string
	ButtonLabels      []string
}

// TryProcessRemovePick handles a 1..99 reply when the latest open disambiguation session is a /remove pick.
// ok is false when the session is not a remove_saved session (caller may try /album pick).
// When ok is true, text is always non-empty.
func (lib *LibraryService) TryProcessRemovePick(ctx context.Context, source, externalID string, oneBased int) (text string, ok bool, err error) {
	if lib == nil || lib.Store == nil {
		return "", false, fmt.Errorf("nil library service")
	}
	sess, raw, err := lib.Store.LatestOpenDisambiguationSession(ctx, source, externalID)
	if err != nil {
		return "", false, err
	}
	if sess == nil || len(raw) == 0 {
		return "", false, nil
	}
	var root removeDisambigRoot
	if jerr := json.Unmarshal(raw, &root); jerr != nil || root.Kind != removeDisambigKind {
		return "", false, nil
	}
	if len(root.Candidates) == 0 {
		_ = lib.Store.DeleteDisambiguationSession(ctx, sess.ID)
		return noActiveRemovePickCopy(), true, nil
	}
	if oneBased < 1 || oneBased > len(root.Candidates) {
		return removePickRangeCopy(len(root.Candidates)), true, nil
	}
	listenerID, err := lib.Store.ListenerIDBySourceExternal(ctx, source, externalID)
	if err != nil {
		return "", true, err
	}
	if listenerID == "" {
		_ = lib.Store.DeleteDisambiguationSession(ctx, sess.ID)
		return noActiveRemovePickCopy(), true, nil
	}
	cand := root.Candidates[oneBased-1]
	deleted, err := lib.Store.DeleteSavedAlbumForListener(ctx, cand.ID, listenerID)
	if err != nil {
		return "", true, err
	}
	if !deleted {
		_ = lib.Store.DeleteDisambiguationSession(ctx, sess.ID)
		return removeNotFoundCopy(), true, nil
	}
	_ = lib.Store.DeleteDisambiguationSession(ctx, sess.ID)
	return "Removed: " + cand.Label, true, nil
}

// HandleRemove processes /remove <query> (call only after [ParseRemoveLine]).
func (lib *LibraryService) HandleRemove(ctx context.Context, source, externalID, userQuery string) (RemoveReply, error) {
	if lib == nil || lib.Store == nil {
		return RemoveReply{}, fmt.Errorf("nil library service")
	}
	q := strings.TrimSpace(userQuery)
	if q == "" {
		return RemoveReply{Text: removeUsageCopy()}, nil
	}
	if utf8.RuneCountInString(q) > MaxQueryRunes {
		return RemoveReply{Text: tooLongQueryCopy()}, nil
	}
	listenerID, err := lib.Store.ListenerIDBySourceExternal(ctx, source, externalID)
	if err != nil {
		return RemoveReply{}, err
	}
	if listenerID == "" {
		return RemoveReply{Text: removeNotFoundCopy()}, nil
	}
	rows, err := lib.Store.ListSavedAlbumRowsForListener(ctx, listenerID)
	if err != nil {
		return RemoveReply{}, err
	}
	nq := NormalizeArtistQuery(q)
	if nq == "" {
		return RemoveReply{Text: removeUsageCopy()}, nil
	}
	exact := exactTitleMatches(rows, nq)
	if len(exact) >= 1 {
		if len(exact) == 1 {
			r := exact[0]
			deleted, err := lib.Store.DeleteSavedAlbumForListener(ctx, r.ID, listenerID)
			if err != nil {
				return RemoveReply{}, err
			}
			if !deleted {
				return RemoveReply{Text: removeNotFoundCopy()}, nil
			}
			return RemoveReply{Text: "Removed: " + FormatSavedAlbumLine(r.Title, r.PrimaryArtist, r.Year)}, nil
		}
		return lib.openRemoveDisambiguation(ctx, listenerID, exact)
	}
	partial := partialTitleMatches(rows, nq)
	if len(partial) == 0 {
		return RemoveReply{Text: removeNotFoundCopy()}, nil
	}
	if len(partial) > 3 {
		return RemoveReply{Text: removeTooManyPartialsCopy()}, nil
	}
	return lib.openRemoveDisambiguation(ctx, listenerID, partial)
}

// exactTitleMatches returns rows where normalized title equals nq.
func exactTitleMatches(rows []store.SavedAlbumListRow, nq string) []store.SavedAlbumListRow {
	if nq == "" {
		return nil
	}
	var out []store.SavedAlbumListRow
	for _, r := range rows {
		if NormalizeArtistQuery(r.Title) == nq {
			out = append(out, r)
		}
	}
	return out
}

// partialTitleMatches returns rows where the normalized title contains nq as a contiguous substring.
// Call only when nq is non-empty and there is no exact match (or after exact tier is empty).
func partialTitleMatches(rows []store.SavedAlbumListRow, nq string) []store.SavedAlbumListRow {
	if nq == "" {
		return nil
	}
	var out []store.SavedAlbumListRow
	for _, r := range rows {
		nt := NormalizeArtistQuery(r.Title)
		if strings.Contains(nt, nq) {
			out = append(out, r)
		}
	}
	return out
}

func (lib *LibraryService) openRemoveDisambiguation(ctx context.Context, listenerID string, matches []store.SavedAlbumListRow) (RemoveReply, error) {
	if err := lib.Store.DeleteDisambigForListener(ctx, listenerID); err != nil {
		return RemoveReply{}, err
	}
	cands := make([]removeDisambigItem, 0, len(matches))
	for _, m := range matches {
		lbl := FormatSavedAlbumLine(m.Title, m.PrimaryArtist, m.Year)
		cands = append(cands, removeDisambigItem{ID: m.ID, Label: lbl})
	}
	payload, err := json.Marshal(removeDisambigRoot{Kind: removeDisambigKind, Candidates: cands})
	if err != nil {
		return RemoveReply{}, err
	}
	sid, err := lib.Store.CreateDisambiguationSession(ctx, listenerID, payload, store.DefaultSessionTTL)
	if err != nil {
		return RemoveReply{}, err
	}
	labels := make([]string, len(cands))
	for i, c := range cands {
		labels[i] = c.Label
	}
	return RemoveReply{
		Text:              formatRemoveDisambigText(len(cands), cands),
		DisambigSessionID: sid,
		ButtonLabels:      labels,
	}, nil
}

func formatRemoveDisambigText(n int, cands []removeDisambigItem) string {
	var b strings.Builder
	b.WriteString("Several albums in your list match. Tap a button or reply with a number 1–")
	b.WriteString(itoa(n))
	b.WriteString(" to remove one:\n")
	for i, c := range cands {
		b.WriteString(itoa(i + 1))
		b.WriteString(": ")
		b.WriteString(c.Label)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func removeUsageCopy() string {
	return "Send which album to remove, for example: /remove the album title, or /remove Kind of Blue"
}

func removeNotFoundCopy() string {
	return "I did not find a saved album in your list that matches that. Try a clearer title or send /list to see what you have saved."
}

func removeTooManyPartialsCopy() string {
	return "That search matches more than 3 saved albums. Send a more specific /remove (more words from the title), or use /list to see your saves."
}

func removePickRangeCopy(n int) string {
	if n <= 0 {
		return "No remove choice is open. Start with /remove and the album title."
	}
	if n == 1 {
		return "Send 1 to pick that row, or start again with /remove and a clearer name."
	}
	return fmt.Sprintf("Send a number from 1 to %d, or start again with /remove.", n)
}

func noActiveRemovePickCopy() string {
	return "No remove choice is open. Use /remove with the album you want to drop from your list."
}
