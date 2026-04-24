package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/mellomaths/lifesoundtrack/bot/internal/store"
)

// SavedAlbumPick is the chosen row for one daily send attempt.
type SavedAlbumPick struct {
	ID              string
	Title           string
	PrimaryArtist   *string
	Year            *int
	ProviderName    string
	ProviderAlbumID *string
	ArtURL          *string
	Extra           []byte
}

// DailyMessenger sends one daily pick to a chat (e.g. Telegram private).
type DailyMessenger interface {
	SendDailyPick(ctx context.Context, chatID int64, msg DailyPickMessage) error
}

// DailyPickMessage is the adapter-facing payload for one send.
type DailyPickMessage struct {
	ArtHTTPSURL  string
	BodyForText  string
	SpotifyURL   string
	UseURLButton bool
}

// PickSavedAlbumForDaily selects one row per FR-003 and the job contract (injectable rand for tests).
func PickSavedAlbumForDaily(rows []store.SavedAlbumForDaily, r *rand.Rand) (SavedAlbumPick, bool) {
	if len(rows) == 0 {
		return SavedAlbumPick{}, false
	}
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	var never []store.SavedAlbumForDaily
	for _, row := range rows {
		if row.LastRecommendedAt == nil {
			never = append(never, row)
		}
	}
	tier := never
	if len(tier) == 0 {
		var minT time.Time
		var have bool
		for _, row := range rows {
			if row.LastRecommendedAt == nil {
				continue
			}
			t := *row.LastRecommendedAt
			if !have || t.Before(minT) {
				minT = t
				have = true
			}
		}
		if !have {
			return SavedAlbumPick{}, false
		}
		for _, row := range rows {
			if row.LastRecommendedAt != nil && row.LastRecommendedAt.Equal(minT) {
				tier = append(tier, row)
			}
		}
	}
	chosen := tier[r.Intn(len(tier))]
	return SavedAlbumPick{
		ID:              chosen.ID,
		Title:           chosen.Title,
		PrimaryArtist:   chosen.PrimaryArtist,
		Year:            chosen.Year,
		ProviderName:    chosen.ProviderName,
		ProviderAlbumID: chosen.ProviderAlbumID,
		ArtURL:          chosen.ArtURL,
		Extra:           chosen.Extra,
	}, true
}

// SpotifyAlbumOpenURL resolves an open.spotify.com URL from saved row fields and optional extra JSON.
func SpotifyAlbumOpenURL(p SavedAlbumPick) string {
	if strings.EqualFold(strings.TrimSpace(p.ProviderName), "spotify") &&
		p.ProviderAlbumID != nil && strings.TrimSpace(*p.ProviderAlbumID) != "" {
		return "https://open.spotify.com/album/" + strings.TrimSpace(*p.ProviderAlbumID)
	}
	if len(p.Extra) > 0 {
		var m map[string]any
		if json.Unmarshal(p.Extra, &m) == nil {
			if u, ok := m["spotify_album_url"].(string); ok && strings.TrimSpace(u) != "" {
				return strings.TrimSpace(u)
			}
		}
	}
	return ""
}

// FormatDailyPickLine builds "Your pick today: TITLE — ARTIST (YEAR)" with year only when set.
func FormatDailyPickLine(title string, artist *string, year *int) string {
	var b strings.Builder
	b.WriteString("Your pick today: ")
	b.WriteString(strings.TrimSpace(title))
	b.WriteString(" — ")
	if artist != nil {
		b.WriteString(strings.TrimSpace(*artist))
	}
	if year != nil {
		fmt.Fprintf(&b, " (%d)", *year)
	}
	return b.String()
}

// DailySignoff is the trailing product line for the daily message.
const DailySignoff = "Enjoy the listen — LifeSoundtrack."

// BuildDailyPickMessage prepares Telegram payload fields from a pick.
func BuildDailyPickMessage(pick SavedAlbumPick) DailyPickMessage {
	spotify := SpotifyAlbumOpenURL(pick)
	line := FormatDailyPickLine(pick.Title, pick.PrimaryArtist, pick.Year)
	useBtn := spotify != ""
	var body string
	if useBtn {
		body = line + "\n\n" + DailySignoff
	} else if spotify != "" {
		body = line + "\n" + spotify + "\n\n" + DailySignoff
	} else {
		body = line + "\n\n" + DailySignoff
	}
	art := ""
	if pick.ArtURL != nil {
		u := strings.TrimSpace(*pick.ArtURL)
		if strings.HasPrefix(strings.ToLower(u), "https://") {
			art = u
		}
	}
	return DailyPickMessage{
		ArtHTTPSURL:  art,
		BodyForText:  body,
		SpotifyURL:   spotify,
		UseURLButton: useBtn,
	}
}

// DailyRecommendAlbumStore is the persistence surface for one daily run (implemented by *store.Store).
type DailyRecommendAlbumStore interface {
	ListSavedAlbumsForDaily(ctx context.Context, listenerID string) ([]store.SavedAlbumForDaily, error)
	RecordRecommendationTx(ctx context.Context, p store.RecordRecommendationParams) error
}

// DailyRecommendRunner loads a pick, sends, then persists on success (FR-007, FR-008).
type DailyRecommendRunner struct {
	Store     DailyRecommendAlbumStore
	Messenger DailyMessenger
	Log       *slog.Logger
	Rand      *rand.Rand
}

// RunForListener processes one listener in a daily run. chatID is the Telegram private chat id.
func (r *DailyRecommendRunner) RunForListener(ctx context.Context, runID, listenerID string, chatID int64) {
	log := r.Log
	if log == nil {
		log = slog.Default()
	}
	rows, err := r.Store.ListSavedAlbumsForDaily(ctx, listenerID)
	if err != nil {
		log.Error("daily_recommendations_albums", "run_id", runID, "listener_id", listenerID, "err", err)
		return
	}
	if len(rows) == 0 {
		log.Info("daily_recommendations_skip", "run_id", runID, "listener_id", listenerID, "reason", "no_saved_albums")
		return
	}
	pick, ok := PickSavedAlbumForDaily(rows, r.Rand)
	if !ok {
		log.Error("daily_recommendations_pick", "run_id", runID, "listener_id", listenerID, "err", "empty_tier")
		return
	}
	msg := BuildDailyPickMessage(pick)
	if r.Messenger == nil {
		log.Error("daily_recommendations_messenger_nil", "run_id", runID, "listener_id", listenerID)
		return
	}
	if err := r.Messenger.SendDailyPick(ctx, chatID, msg); err != nil {
		log.Warn("daily_recommendations_send_failed", "run_id", runID, "listener_id", listenerID, "saved_album_id", pick.ID, "err", err)
		return
	}
	sentAt := time.Now().UTC()
	spotify := SpotifyAlbumOpenURL(pick)
	var spotifyPtr *string
	if spotify != "" {
		spotifyPtr = &spotify
	}
	if err := r.Store.RecordRecommendationTx(ctx, store.RecordRecommendationParams{
		RunID:              runID,
		ListenerID:         listenerID,
		SavedAlbumID:       pick.ID,
		TitleSnapshot:      pick.Title,
		ArtistSnapshot:     pick.PrimaryArtist,
		YearSnapshot:       pick.Year,
		SpotifyURLSnapshot: spotifyPtr,
		SentAt:             sentAt,
	}); err != nil {
		log.Error("daily_recommendations_persist_failed", "run_id", runID, "listener_id", listenerID, "saved_album_id", pick.ID, "err", err)
		return
	}
	log.Info("daily_recommendations_sent", "run_id", runID, "listener_id", listenerID, "saved_album_id", pick.ID)
}
