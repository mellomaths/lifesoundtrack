package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"unicode/utf8"

	"github.com/mellomaths/lifesoundtrack/bot/internal/store"
)

// MaxQueryRunes is the free-form /album text cap (contracts/album-command.md).
const MaxQueryRunes = 512

// Outcome is the public state after handling an album line or pick.
type Outcome int

const (
	OutcomeEmptyQuery Outcome = iota
	OutcomeTooLong
	OutcomeNoMatch
	OutcomeProviderExhausted
	OutcomeTransientError
	OutcomeDisambig
	OutcomeSaved
	OutcomePickOutOfRange
	OutcomeNoSession
)

// UserMessage is non-technical user-visible text (FR-007).
type UserMessage struct {
	Outcome Outcome
	Text    string
	// PickCount is 1–3 when Outcome == OutcomeDisambig (inline keyboard size).
	PickCount int
}

// SaveService wires metadata search to persistence.
type SaveService struct {
	Store  *store.Store
	Search MetadataOrchestrator
	Log    *slog.Logger
}

// ProcessAlbumQuery handles a non-pick /album line (US1, US1b, US2).
func (s *SaveService) ProcessAlbumQuery(ctx context.Context, source, externalID, displayName, username, userQuery string) (UserMessage, error) {
	q := userQuery
	if q == "" {
		return UserMessage{Outcome: OutcomeEmptyQuery, Text: emptyAlbumQueryCopy()}, nil
	}
	if utf8.RuneCountInString(q) > MaxQueryRunes {
		return UserMessage{Outcome: OutcomeTooLong, Text: tooLongQueryCopy()}, nil
	}
	listener, err := s.Store.UpsertListener(ctx, source, externalID, displayName, username)
	if err != nil {
		return UserMessage{}, err
	}
	cands, err := s.Search.Search(ctx, q)
	if err != nil {
		if errors.Is(err, ErrNoMatch) {
			return UserMessage{Outcome: OutcomeNoMatch, Text: noMatchCopy()}, nil
		}
		if errors.Is(err, ErrAllProvidersExhausted) {
			return UserMessage{Outcome: OutcomeProviderExhausted, Text: tryAgainCopy()}, nil
		}
		if s.Log != nil {
			s.Log.Warn("metadata search failed", "class", "transient", "err", err)
		}
		return UserMessage{Outcome: OutcomeTransientError, Text: tryAgainCopy()}, nil
	}
	if len(cands) == 0 {
		return UserMessage{Outcome: OutcomeNoMatch, Text: noMatchCopy()}, nil
	}
	if len(cands) == 1 {
		_, ums, err := s.persistSave(ctx, listener.ID, &cands[0], q, cands[0].Provider, cands[0].ProviderRef, cands[0].Title, cands[0].PrimaryArtist, cands[0].Year, cands[0].Genres, cands[0].ArtURL, nil)
		if err != nil {
			return UserMessage{}, err
		}
		return ums, nil
	}
	// 2+ candidates: up to 3, store session, return disambig
	top := cands
	if len(top) > 3 {
		top = top[:3]
	}
	if err := s.Store.DeleteDisambigForListener(ctx, listener.ID); err != nil {
		return UserMessage{}, err
	}
	b, err := json.Marshal(top)
	if err != nil {
		return UserMessage{}, err
	}
	if _, err := s.Store.CreateDisambiguationSession(ctx, listener.ID, b, store.DefaultSessionTTL); err != nil {
		return UserMessage{}, err
	}
	lines := formatCandidateLines(top, true)
	return UserMessage{
		Outcome:   OutcomeDisambig,
		Text:      "Pick the album (buttons or send 1–" + itoa(len(top)) + "):\n" + lines,
		PickCount: len(top),
	}, nil
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

// ProcessPickByIndex records a 1-based index from the open disambiguation session (US1b).
func (s *SaveService) ProcessPickByIndex(ctx context.Context, source, externalID, displayName, username string, oneBased int) (UserMessage, error) {
	listener, err := s.Store.UpsertListener(ctx, source, externalID, displayName, username)
	if err != nil {
		return UserMessage{}, err
	}
	sess, raw, err := s.Store.LatestOpenDisambiguationSession(ctx, source, externalID)
	if err != nil {
		return UserMessage{}, err
	}
	if sess == nil || len(raw) == 0 {
		return UserMessage{Outcome: OutcomeNoSession, Text: noActivePickCopy()}, nil
	}
	var cands []AlbumCandidate
	if err := json.Unmarshal(raw, &cands); err != nil {
		return UserMessage{}, err
	}
	if oneBased < 1 || oneBased > len(cands) || oneBased > 3 {
		return UserMessage{Outcome: OutcomePickOutOfRange, Text: pickRangeCopy(len(cands))}, nil
	}
	c := cands[oneBased-1]
	_, ums, err := s.persistSave(ctx, listener.ID, &c, "", c.Provider, c.ProviderRef, c.Title, c.PrimaryArtist, c.Year, c.Genres, c.ArtURL, &sess.ID)
	if err != nil {
		return UserMessage{}, err
	}
	return ums, nil
}

func (s *SaveService) persistSave(
	ctx context.Context, listenerID string, c *AlbumCandidate,
	userQuery, prov, provRef, title, artist string, year *int, genres []string, art string, disambigID *string,
) (string, UserMessage, error) {
	extra, err := EncodeCandidateExtra(c)
	if err != nil {
		return "", UserMessage{}, err
	}
	savedID, err := s.Store.InsertSavedAlbum(ctx, store.InsertSavedAlbumParams{
		ListenerID:        listenerID,
		UserQueryText:     strOrNil(userQuery),
		Title:             title,
		PrimaryArtist:     strOrNil(artist),
		Year:              year,
		Genres:            CapGenres(genres),
		ProviderName:      prov,
		ProviderAlbumID:   strOrNil(provRef),
		ArtURL:            strOrNil(art),
		Extra:             extra,
	})
	if err != nil {
		return "", UserMessage{}, err
	}
	if disambigID != nil {
		_ = s.Store.DeleteDisambiguationSession(ctx, *disambigID)
	}
	_ = savedID
	yr := yearStr(year)
	confirm := "Saved: " + title
	if p := strOrNil(artist); p != nil {
		confirm += " — " + *p
	}
	if yr != "" {
		confirm += " (" + yr + ")"
	}
	return savedID, UserMessage{Outcome: OutcomeSaved, Text: confirm}, nil
}

func yearStr(y *int) string {
	if y == nil {
		return ""
	}
	return itoa(*y)
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func formatCandidateLines(cands []AlbumCandidate, numbered bool) string {
	var b string
	for i, c := range cands {
		line := c.Title
		if c.PrimaryArtist != "" {
			line += " — " + c.PrimaryArtist
		}
		if c.Year != nil {
			line += fmt.Sprintf(" (%d)", *c.Year)
		}
		if numbered {
			b += itoa(i+1) + ": " + line + "\n"
		} else {
			b += line + "\n"
		}
	}
	return b
}

func emptyAlbumQueryCopy() string {
	return "Add what to look for: /album <search text> (e.g. album title or artist)."
}

func tooLongQueryCopy() string {
	return "That search is too long. Try a shorter line (up to 512 characters)."
}

func noMatchCopy() string {
	return "I could not find a matching album. Try different words or check spelling."
}

func tryAgainCopy() string {
	return "Music lookup is not available right now. Please try again in a few minutes."
}

func noActivePickCopy() string {
	return "No album choice is open. Start with /album and your search text."
}

func pickRangeCopy(n int) string {
	if n <= 1 {
		return "Send 1 to pick the album, or start a new search with /album."
	}
	return fmt.Sprintf("Send a number from 1 to %d, or use the buttons.", n)
}
