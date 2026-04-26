package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mellomaths/lifesoundtrack/bot/internal/store"
)

// MaxQueryRunes is the free-form /album text cap (contracts/album-command.md).
const MaxQueryRunes = 512

// SavePersistence is the store surface needed by [SaveService] (satisfied by *store.Store; mock in tests).
type SavePersistence interface {
	UpsertListener(ctx context.Context, source, externalID, displayName, username string) (*store.Listener, error)
	DeleteDisambigForListener(ctx context.Context, listenerID string) error
	CreateDisambiguationSession(ctx context.Context, listenerID string, candidatesJSON []byte, ttl time.Duration) (string, error)
	InsertSavedAlbum(ctx context.Context, p store.InsertSavedAlbumParams) (id string, err error)
	LatestOpenDisambiguationSession(ctx context.Context, source, externalID string) (*store.Session, []byte, error)
	DeleteDisambiguationSession(ctx context.Context, sessionID string) error
}

// Outcome is the public state after handling an album line or pick.
type Outcome int

const (
	OutcomeEmptyQuery Outcome = iota
	OutcomeTooLong
	OutcomeNoMatch
	OutcomeProviderExhausted
	OutcomeTransientError
	OutcomeDisambig
	OutcomeRefineQuery
	OutcomeSaved
	OutcomePickOutOfRange
	OutcomeNoSession
	OutcomeBadSpotifyLink
	OutcomeMultiSpotifyLink
)

// UserMessage is non-technical user-visible text (FR-007).
type UserMessage struct {
	Outcome Outcome
	Text    string
	// PickCount is 1 or 2 when Outcome == OutcomeDisambig (album rows; a separate "Other" control is UI-only).
	PickCount int
	// AlbumButtonLabels is one label per album row: "ALBUM_TITLE | ARTIST (YEAR)" (used by Telegram; max 2).
	AlbumButtonLabels []string
}

// SaveService wires metadata search to persistence.
type SaveService struct {
	Store  SavePersistence
	Search MetadataOrchestrator
	Log    *slog.Logger
}

// trySpotifyDirectAlbumPath handles FR-008 Spotify album / share URLs before free-text Search.
func (s *SaveService) trySpotifyDirectAlbumPath(ctx context.Context, listenerID, q string) (UserMessage, bool, error) {
	plan := PlanSpotifyAlbumQuery(q)
	switch plan.Mode {
	case SpotifyModeNone:
		return UserMessage{}, false, nil
	case SpotifyModeAmbiguousMulti:
		if s.Log != nil {
			s.Log.Info("album query classified", "spotify_path", "multi_link")
		}
		return UserMessage{Outcome: OutcomeMultiSpotifyLink, Text: multiSpotifyLinkCopy()}, true, nil
	case SpotifyModeIneligibleSpotifyHost:
		if s.Log != nil {
			s.Log.Info("album query classified", "spotify_path", "ineligible_spotify_page")
		}
		return UserMessage{Outcome: OutcomeBadSpotifyLink, Text: badSpotifyLinkCopy()}, true, nil
	case SpotifyModeDirect:
		return s.finishSpotifyAlbumByID(ctx, listenerID, q, plan.AlbumID)
	case SpotifyModeResolveShort:
		id, err := s.Search.ResolveSpotifyShareURL(ctx, plan.ShareURL)
		if err != nil {
			if s.Log != nil {
				s.Log.Warn("spotify share resolve failed", "spotify_path", "resolve", "err", err)
			}
			if errors.Is(err, ErrAllProvidersExhausted) {
				return UserMessage{Outcome: OutcomeProviderExhausted, Text: tryAgainCopy()}, true, nil
			}
			return UserMessage{Outcome: OutcomeBadSpotifyLink, Text: badSpotifyLinkCopy()}, true, nil
		}
		return s.finishSpotifyAlbumByID(ctx, listenerID, q, id)
	default:
		return UserMessage{}, false, nil
	}
}

func (s *SaveService) finishSpotifyAlbumByID(ctx context.Context, listenerID, userQuery, albumID string) (UserMessage, bool, error) {
	cands, err := s.Search.LookupSpotifyAlbumByID(ctx, albumID)
	if err != nil {
		if errors.Is(err, ErrAllProvidersExhausted) {
			return UserMessage{Outcome: OutcomeProviderExhausted, Text: tryAgainCopy()}, true, nil
		}
		if errors.Is(err, ErrNoMatch) {
			return UserMessage{Outcome: OutcomeNoMatch, Text: noMatchCopy()}, true, nil
		}
		if s.Log != nil {
			s.Log.Warn("spotify album by id failed", "spotify_path", "lookup", "err", err)
		}
		return UserMessage{Outcome: OutcomeTransientError, Text: tryAgainCopy()}, true, nil
	}
	if len(cands) != 1 {
		return UserMessage{Outcome: OutcomeNoMatch, Text: noMatchCopy()}, true, nil
	}
	if s.Log != nil {
		s.Log.Info("spotify direct album resolved", "spotify_path", "album_id_lookup", "outcome", "ok")
	}
	_, ums, err := s.persistSave(ctx, listenerID, &cands[0], userQuery, cands[0].Provider, cands[0].ProviderRef, cands[0].Title, cands[0].PrimaryArtist, cands[0].Year, cands[0].Genres, cands[0].ArtURL, nil)
	if err != nil {
		return UserMessage{}, true, err
	}
	return ums, true, nil
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

	if um, ok, err := s.trySpotifyDirectAlbumPath(ctx, listener.ID, q); ok {
		return um, err
	} else if err != nil {
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
	cands = dedupeCandidatesByAlbumLine(cands)
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
	// 2+ distinct user-visible labels: offer top 2 by relevance; "Other" is a separate UI path (no third album row in JSON).
	top := cands
	if len(top) > 2 {
		top = top[:2]
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
	labels := albumButtonLabels(top)
	return UserMessage{
		Outcome:           OutcomeDisambig,
		Text:              "Pick the album (buttons, or send 1 or 2; send 3 for Other if none match):\n" + lines,
		PickCount:         len(top),
		AlbumButtonLabels: labels,
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
	// 3 = "Other" (only when two album rows are shown).
	if oneBased == 3 {
		if len(cands) == 2 {
			_ = s.Store.DeleteDisambiguationSession(ctx, sess.ID)
			return UserMessage{Outcome: OutcomeRefineQuery, Text: refineQueryCopy()}, nil
		}
		return UserMessage{Outcome: OutcomePickOutOfRange, Text: pickRangeCopy(len(cands))}, nil
	}
	if oneBased < 1 || oneBased > len(cands) || oneBased > 2 {
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
	// Single-result and Spotify-direct saves are not part of the active [AlbumCandidate] disambig
	// session. Clear any stale sessions (e.g. a prior /remove pick list) so a later "1" message
	// cannot be interpreted as a remove.
	if disambigID == nil {
		if err := s.Store.DeleteDisambigForListener(ctx, listenerID); err != nil {
			return "", UserMessage{}, err
		}
	}
	savedID, err := s.Store.InsertSavedAlbum(ctx, store.InsertSavedAlbumParams{
		ListenerID:      listenerID,
		UserQueryText:   strOrNil(userQuery),
		Title:           title,
		PrimaryArtist:   strOrNil(artist),
		Year:            year,
		Genres:          CapGenres(genres),
		ProviderName:    prov,
		ProviderAlbumID: strOrNil(provRef),
		ArtURL:          strOrNil(art),
		Extra:           extra,
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

// dedupeCandidatesByAlbumLine keeps the first candidate for each user-visible
// "ALBUM_TITLE | ARTIST (YEAR)" string (per formatAlbumLine), i.e. first in
// relevance order among equivalent rows.
func dedupeCandidatesByAlbumLine(cands []AlbumCandidate) []AlbumCandidate {
	if len(cands) < 2 {
		return cands
	}
	seen := make(map[string]struct{}, len(cands))
	out := make([]AlbumCandidate, 0, len(cands))
	for _, c := range cands {
		key := formatAlbumLine(c)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}
	return out
}

// FormatSavedAlbumLine formats a saved row like disambiguation labels (Title | Artist (Year)).
func FormatSavedAlbumLine(title string, primaryArtist *string, year *int) string {
	c := AlbumCandidate{Title: title, Year: year}
	if primaryArtist != nil {
		c.PrimaryArtist = *primaryArtist
	}
	return formatAlbumLine(c)
}

func formatAlbumLine(c AlbumCandidate) string {
	var b strings.Builder
	b.WriteString(c.Title)
	if c.PrimaryArtist != "" {
		b.WriteString(" | ")
		b.WriteString(c.PrimaryArtist)
	}
	if c.Year != nil {
		b.WriteString(" (")
		b.WriteString(itoa(*c.Year))
		b.WriteString(")")
	}
	return b.String()
}

func albumButtonLabels(cands []AlbumCandidate) []string {
	labels := make([]string, 0, len(cands))
	for _, c := range cands {
		labels = append(labels, formatAlbumLine(c))
	}
	return labels
}

func formatCandidateLines(cands []AlbumCandidate, numbered bool) string {
	var b string
	for i, c := range cands {
		line := formatAlbumLine(c)
		if numbered {
			b += itoa(i+1) + ": " + line + "\n"
		} else {
			b += line + "\n"
		}
	}
	return b
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
	if n <= 0 {
		return "Start a new search with /album."
	}
	if n == 1 {
		return "Send 1 to pick the album, or start a new search with /album."
	}
	if n == 2 {
		return "Send 1 or 2 to pick an album, 3 for Other, or use the buttons."
	}
	return fmt.Sprintf("Send a number from 1 to %d, or use the buttons.", n)
}

func refineQueryCopy() string {
	return "Try a more specific /album line: include the full album title, the artist name, and the release year (e.g. the year on the cover)."
}
