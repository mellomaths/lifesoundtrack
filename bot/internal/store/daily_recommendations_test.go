package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestListTelegramListenerIDsWithSavedAlbums_Postgres runs the listener enumeration query against
// PostgreSQL (SC-007, FR-013–FR-015). Skips when DATABASE_URL is unset (e.g. CI without service container).
func TestListTelegramListenerIDsWithSavedAlbums_Postgres(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skip Postgres integration test")
	}
	ctx := context.Background()
	migDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(migDir, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(st.Close)

	extA := "test-daily-" + uuid.NewString()
	extB := "test-daily-" + uuid.NewString()
	extC := "test-daily-" + uuid.NewString()

	la, err := st.UpsertListener(ctx, ListenerSourceTelegram, extA, "", "")
	if err != nil {
		t.Fatal(err)
	}
	lb, err := st.UpsertListener(ctx, ListenerSourceTelegram, extB, "", "")
	if err != nil {
		t.Fatal(err)
	}
	lc, err := st.UpsertListener(ctx, "other", extC, "", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, id := range []string{la.ID, lb.ID, lc.ID} {
			_, _ = st.pool.Exec(ctx, `DELETE FROM saved_albums WHERE listener_id = $1`, id)
			_, _ = st.pool.Exec(ctx, `DELETE FROM listeners WHERE id = $1`, id)
		}
	})

	_, err = st.InsertSavedAlbum(ctx, InsertSavedAlbumParams{
		ListenerID:   la.ID,
		Title:        "Album A",
		ProviderName: "spotify",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.InsertSavedAlbum(ctx, InsertSavedAlbumParams{
		ListenerID:   la.ID,
		Title:        "Album A2",
		ProviderName: "spotify",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.InsertSavedAlbum(ctx, InsertSavedAlbumParams{
		ListenerID:   lb.ID,
		Title:        "Album B",
		ProviderName: "spotify",
	})
	if err != nil {
		t.Fatal(err)
	}
	// lc has no saved albums — should not appear.

	ids, err := st.ListTelegramListenerIDsWithSavedAlbums(ctx)
	if err != nil {
		t.Fatalf("list listeners: %v", err)
	}
	seen := make(map[string]int, len(ids))
	for _, id := range ids {
		seen[id]++
	}
	for id, n := range seen {
		if n != 1 {
			t.Fatalf("listener %s appears %d times; want at most once per run", id, n)
		}
	}
	want := map[string]struct{}{la.ID: {}, lb.ID: {}}
	for id := range want {
		if seen[id] != 1 {
			t.Fatalf("expected id %s in result, got %#v", id, ids)
		}
	}
	if seen[lc.ID] != 0 {
		t.Fatalf("listener without albums should not be listed; got %v", ids)
	}
	idx := func(target string) int {
		for i, id := range ids {
			if id == target {
				return i
			}
		}
		return -1
	}
	ia, ib := idx(la.ID), idx(lb.ID)
	if ia < 0 || ib < 0 {
		t.Fatalf("expected seeded listeners in result; got %v", ids)
	}
	if ia >= ib {
		t.Fatalf("expected listener created first (%s) before second (%s) in ORDER BY created_at; got %v", la.ID, lb.ID, ids)
	}
}

// TestRecordRecommendationTx_Postgres exercises last_recommended_at + recommendations insert (FR-007, T007).
func TestRecordRecommendationTx_Postgres(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skip Postgres integration test")
	}
	ctx := context.Background()
	migDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(migDir, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(st.Close)

	ext := "test-rec-tx-" + uuid.NewString()
	l, err := st.UpsertListener(ctx, ListenerSourceTelegram, ext, "", "")
	if err != nil {
		t.Fatal(err)
	}
	savedID, err := st.InsertSavedAlbum(ctx, InsertSavedAlbumParams{
		ListenerID:   l.ID,
		Title:        "Tx Album",
		PrimaryArtist: strPtr("Artist"),
		ProviderName: "spotify",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = st.pool.Exec(ctx, `DELETE FROM recommendations WHERE listener_id = $1::uuid`, l.ID)
		_, _ = st.pool.Exec(ctx, `DELETE FROM saved_albums WHERE listener_id = $1::uuid`, l.ID)
		_, _ = st.pool.Exec(ctx, `DELETE FROM listeners WHERE id = $1::uuid`, l.ID)
	})

	runID := uuid.NewString()
	sent := time.Date(2026, 4, 24, 6, 0, 0, 0, time.UTC)
	artist := "Artist"
	year := 2020
	spotify := "https://open.spotify.com/album/abc"
	if err := st.RecordRecommendationTx(ctx, RecordRecommendationParams{
		RunID:              runID,
		ListenerID:         l.ID,
		SavedAlbumID:       savedID,
		TitleSnapshot:      "Tx Album",
		ArtistSnapshot:     &artist,
		YearSnapshot:       &year,
		SpotifyURLSnapshot: &spotify,
		SentAt:             sent,
	}); err != nil {
		t.Fatalf("RecordRecommendationTx: %v", err)
	}

	var last sql.NullTime
	if err := st.pool.QueryRow(ctx, `SELECT last_recommended_at FROM saved_albums WHERE id = $1::uuid`, savedID).Scan(&last); err != nil {
		t.Fatal(err)
	}
	if !last.Valid || !last.Time.Equal(sent) {
		t.Fatalf("last_recommended_at = %v, want %v", last, sent)
	}

	var count int
	if err := st.pool.QueryRow(ctx,
		`SELECT count(*) FROM recommendations WHERE listener_id = $1::uuid AND run_id = $2::uuid`,
		l.ID, runID,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("recommendations rows: %d", count)
	}

	err = st.RecordRecommendationTx(ctx, RecordRecommendationParams{
		RunID:         runID,
		ListenerID:    l.ID,
		SavedAlbumID:  savedID,
		TitleSnapshot: "Dup",
		SentAt:        sent,
	})
	if err == nil {
		t.Fatal("expected duplicate (listener_id, run_id) to fail")
	}
}

func strPtr(s string) *string { return &s }
