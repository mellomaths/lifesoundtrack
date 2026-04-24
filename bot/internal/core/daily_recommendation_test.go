package core

import (
	"context"
	"io"
	"log/slog"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/mellomaths/lifesoundtrack/bot/internal/store"
)

func TestPickSavedAlbumForDaily_neverRecommendedWins(t *testing.T) {
	t1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := []store.SavedAlbumForDaily{
		{ID: "a", Title: "Old", LastRecommendedAt: &t1},
		{ID: "b", Title: "New", LastRecommendedAt: nil},
	}
	r := rand.New(rand.NewSource(42))
	pick, ok := PickSavedAlbumForDaily(rows, r)
	if !ok || pick.ID != "b" {
		t.Fatalf("got %+v ok=%v", pick, ok)
	}
}

func TestPickSavedAlbumForDaily_oldestTimestampWins(t *testing.T) {
	tOld := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	tNew := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := []store.SavedAlbumForDaily{
		{ID: "a", Title: "Mid", LastRecommendedAt: &tNew},
		{ID: "b", Title: "First", LastRecommendedAt: &tOld},
	}
	r := rand.New(rand.NewSource(1))
	pick, ok := PickSavedAlbumForDaily(rows, r)
	if !ok || pick.ID != "b" {
		t.Fatalf("got %+v ok=%v", pick, ok)
	}
}

func TestPickSavedAlbumForDaily_tieRandom(t *testing.T) {
	rows := []store.SavedAlbumForDaily{
		{ID: "x", Title: "X", LastRecommendedAt: nil},
		{ID: "y", Title: "Y", LastRecommendedAt: nil},
	}
	seen := map[string]struct{}{}
	r := rand.New(rand.NewSource(99))
	for i := 0; i < 40; i++ {
		pick, ok := PickSavedAlbumForDaily(rows, r)
		if !ok {
			t.Fatal("expected ok")
		}
		seen[pick.ID] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatalf("expected both ids over draws; got %v", seen)
	}
}

func TestSpotifyAlbumOpenURL(t *testing.T) {
	id := "abc123"
	p := SavedAlbumPick{ProviderName: "spotify", ProviderAlbumID: &id}
	if u := SpotifyAlbumOpenURL(p); u != "https://open.spotify.com/album/abc123" {
		t.Fatalf("got %q", u)
	}
	p2 := SavedAlbumPick{ProviderName: "spotify", Extra: []byte(`{"spotify_album_url":"https://open.spotify.com/album/x"}`)}
	if u := SpotifyAlbumOpenURL(p2); u != "https://open.spotify.com/album/x" {
		t.Fatalf("got %q", u)
	}
}

func TestFormatDailyPickLine(t *testing.T) {
	artist := "Artist"
	year := 1999
	s := FormatDailyPickLine("Title", &artist, &year)
	if !strings.Contains(s, "Title") || !strings.Contains(s, "Artist") || !strings.Contains(s, "1999") {
		t.Fatalf("got %q", s)
	}
}

func TestBuildDailyPickMessage_URLButtonOmitsURLFromBody(t *testing.T) {
	id := "z"
	p := SavedAlbumPick{
		Title: "T", PrimaryArtist: strPtr("A"), ProviderName: "spotify", ProviderAlbumID: &id,
	}
	m := BuildDailyPickMessage(p)
	if !m.UseURLButton || m.SpotifyURL == "" {
		t.Fatalf("%+v", m)
	}
	if strings.Contains(m.BodyForText, "open.spotify.com") {
		t.Fatalf("body should not contain URL when using button: %q", m.BodyForText)
	}
	if !strings.Contains(m.BodyForText, DailySignoff) {
		t.Fatalf("missing signoff: %q", m.BodyForText)
	}
}

func strPtr(s string) *string { return &s }

type stubDailyStore struct {
	rows []store.SavedAlbumForDaily
	err  error
	rec  *store.RecordRecommendationParams
}

func (s *stubDailyStore) ListSavedAlbumsForDaily(ctx context.Context, listenerID string) ([]store.SavedAlbumForDaily, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rows, nil
}

func (s *stubDailyStore) RecordRecommendationTx(ctx context.Context, p store.RecordRecommendationParams) error {
	s.rec = &p
	return nil
}

type stubMessenger struct {
	err error
	got *DailyPickMessage
}

func (m *stubMessenger) SendDailyPick(ctx context.Context, chatID int64, msg DailyPickMessage) error {
	if m.got != nil {
		*m.got = msg
	}
	return m.err
}

func TestDailyRecommendRunner_sendErrorSkipsPersist(t *testing.T) {
	prov := "x"
	st := &stubDailyStore{rows: []store.SavedAlbumForDaily{{
		ID: "1", Title: "T", ProviderName: "spotify", ProviderAlbumID: &prov,
	}}}
	ms := &stubMessenger{err: context.Canceled}
	r := DailyRecommendRunner{
		Store:     st,
		Messenger: ms,
		Log:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Rand:      rand.New(rand.NewSource(1)),
	}
	r.RunForListener(context.Background(), "550e8400-e29b-41d4-a716-446655440000", "listener", 42)
	if st.rec != nil {
		t.Fatalf("expected no persist on send error; got %+v", st.rec)
	}
}

func TestDailyRecommendRunner_successPersists(t *testing.T) {
	prov := "ab"
	st := &stubDailyStore{rows: []store.SavedAlbumForDaily{{
		ID: "1", Title: "T", PrimaryArtist: strPtr("A"), ProviderName: "spotify", ProviderAlbumID: &prov,
	}}}
	ms := &stubMessenger{}
	r := DailyRecommendRunner{
		Store:     st,
		Messenger: ms,
		Log:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Rand:      rand.New(rand.NewSource(1)),
	}
	r.RunForListener(context.Background(), "550e8400-e29b-41d4-a716-446655440001", "660e8400-e29b-41d4-a716-446655440002", 99)
	if st.rec == nil {
		t.Fatal("expected RecordRecommendationTx")
	}
	if st.rec.SavedAlbumID != "1" || st.rec.ListenerID != "660e8400-e29b-41d4-a716-446655440002" {
		t.Fatalf("bad params %+v", st.rec)
	}
}
