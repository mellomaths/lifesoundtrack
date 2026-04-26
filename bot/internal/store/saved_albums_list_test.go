package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestListSavedAlbums_normalizationVariantsAndIsolation runs list queries against Postgres when DATABASE_URL is set.
func TestListSavedAlbums_normalizationVariantsAndIsolation(t *testing.T) {
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

	extA := "test-list-a-" + uuid.NewString()
	extB := "test-list-b-" + uuid.NewString()
	la, err := st.UpsertListener(ctx, ListenerSourceTelegram, extA, "", "")
	if err != nil {
		t.Fatal(err)
	}
	lb, err := st.UpsertListener(ctx, ListenerSourceTelegram, extB, "", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, id := range []string{la.ID, lb.ID} {
			_, _ = st.pool.Exec(ctx, `DELETE FROM album_list_sessions WHERE listener_id = $1::uuid`, id)
			_, _ = st.pool.Exec(ctx, `DELETE FROM saved_albums WHERE listener_id = $1::uuid`, id)
			_, _ = st.pool.Exec(ctx, `DELETE FROM listeners WHERE id = $1::uuid`, id)
		}
	})

	beatles := "The Beatles"
	_, err = st.InsertSavedAlbum(ctx, InsertSavedAlbumParams{
		ListenerID:    la.ID,
		Title:         "Abbey Road",
		PrimaryArtist: &beatles,
		Year:          nil,
		ProviderName:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	other := "Radiohead"
	_, err = st.InsertSavedAlbum(ctx, InsertSavedAlbumParams{
		ListenerID:    la.ID,
		Title:         "OK Computer",
		PrimaryArtist: &other,
		Year:          nil,
		ProviderName:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	secret := "Secret Artist"
	_, err = st.InsertSavedAlbum(ctx, InsertSavedAlbumParams{
		ListenerID:    lb.ID,
		Title:         "Other Listener",
		PrimaryArtist: &secret,
		Year:          nil,
		ProviderName:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	needles := []string{"beatles", "  BEATLES  ", "the beatles"}
	var firstIDs []string
	for _, n := range needles {
		rows, err := st.ListSavedAlbumsPage(ctx, la.ID, &n, 0, 10)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(rows) != 1 || rows[0].Title != "Abbey Road" {
			t.Fatalf("needle %q: got %+v", n, rows)
		}
		firstIDs = append(firstIDs, rows[0].ID)
	}
	if firstIDs[0] != firstIDs[1] || firstIDs[1] != firstIDs[2] {
		t.Fatalf("expected same album id for normalization variants, got %v", firstIDs)
	}

	n := "beatles"
	countB, err := st.CountSavedAlbumsForListener(ctx, lb.ID, &n)
	if err != nil {
		t.Fatal(err)
	}
	if countB != 0 {
		t.Fatalf("listener B should have 0 beatles matches, got %d", countB)
	}

	sid, err := st.InsertAlbumListSession(ctx, la.ID, &n, 1, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := st.GetAlbumListSession(ctx, sid)
	if err != nil || sess == nil || sess.ListenerID != la.ID {
		t.Fatalf("session: %+v err %v", sess, err)
	}
}
