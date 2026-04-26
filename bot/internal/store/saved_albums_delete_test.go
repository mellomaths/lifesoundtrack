package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

// TestDeleteSavedAlbumForListener_isolation ensures another listener's UUID cannot delete rows.
func TestDeleteSavedAlbumForListener_isolation(t *testing.T) {
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

	extA := "test-del-a-" + uuid.NewString()
	extB := "test-del-b-" + uuid.NewString()
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
			_, _ = st.pool.Exec(ctx, `DELETE FROM saved_albums WHERE listener_id = $1::uuid`, id)
			_, _ = st.pool.Exec(ctx, `DELETE FROM listeners WHERE id = $1::uuid`, id)
		}
	})

	artist := "Artist"
	albumID, err := st.InsertSavedAlbum(ctx, InsertSavedAlbumParams{
		ListenerID:    la.ID,
		Title:         "Target",
		PrimaryArtist: &artist,
		ProviderName:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	all, err := st.ListSavedAlbumRowsForListener(ctx, la.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != albumID || all[0].Title != "Target" {
		t.Fatalf("ListSavedAlbumRowsForListener: %+v", all)
	}

	deleted, err := st.DeleteSavedAlbumForListener(ctx, albumID, lb.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted {
		t.Fatal("must not delete another listener's row")
	}

	n, err := st.DeleteSavedAlbumForListener(ctx, albumID, la.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !n {
		t.Fatal("expected delete for own listener")
	}
}
