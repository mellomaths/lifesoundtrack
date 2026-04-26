package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/mellomaths/lifesoundtrack/bot/internal/store"
)

func TestTryProcessRemovePick_callbackUsesKeyedSessionNotLatest(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skip Postgres integration test")
	}
	ctx := context.Background()
	migDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RunMigrations(migDir, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := store.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(st.Close)

	ext := "test-rmp-keyed-" + uuid.NewString()
	listener, err := st.UpsertListener(ctx, store.ListenerSourceTelegram, ext, "", "")
	if err != nil {
		t.Fatal(err)
	}
	artist := "Artist"
	idA, err := st.InsertSavedAlbum(ctx, store.InsertSavedAlbumParams{
		ListenerID:    listener.ID,
		Title:         "Alpha",
		PrimaryArtist: &artist,
		ProviderName:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	idB, err := st.InsertSavedAlbum(ctx, store.InsertSavedAlbumParams{
		ListenerID:    listener.ID,
		Title:         "Beta",
		PrimaryArtist: &artist,
		ProviderName:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = st.DeleteListener(ctx, listener.ID)
	})

	payloadOld, err := json.Marshal(removeDisambigRoot{Kind: removeDisambigKind, Candidates: []removeDisambigItem{
		{ID: idA, Label: "Alpha"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	payloadNew, err := json.Marshal(removeDisambigRoot{Kind: removeDisambigKind, Candidates: []removeDisambigItem{
		{ID: idB, Label: "Beta"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	sidOld, err := st.CreateDisambiguationSession(ctx, listener.ID, payloadOld, store.DefaultSessionTTL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.CreateDisambiguationSession(ctx, listener.ID, payloadNew, store.DefaultSessionTTL)
	if err != nil {
		t.Fatal(err)
	}

	lib := &LibraryService{Store: st}
	msg, ok, err := lib.TryProcessRemovePick(ctx, store.ListenerSourceTelegram, ext, sidOld, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || msg == "" {
		t.Fatalf("handled=%v msg=%q", ok, msg)
	}

	rows, err := st.ListSavedAlbumRowsForListener(ctx, listener.ID)
	if err != nil {
		t.Fatal(err)
	}
	var hasA, hasB bool
	for _, r := range rows {
		switch r.ID {
		case idA:
			hasA = true
		case idB:
			hasB = true
		}
	}
	if hasA {
		t.Fatal("Alpha should have been removed via keyed old session")
	}
	if !hasB {
		t.Fatal("Beta should remain (latest session must not drive this callback)")
	}
}

func TestTryProcessRemovePick_staleKeyedSessionDoesNotDelete(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skip Postgres integration test")
	}
	ctx := context.Background()
	migDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RunMigrations(migDir, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := store.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(st.Close)

	ext := "test-rmp-stale-" + uuid.NewString()
	listener, err := st.UpsertListener(ctx, store.ListenerSourceTelegram, ext, "", "")
	if err != nil {
		t.Fatal(err)
	}
	artist := "Artist"
	idB, err := st.InsertSavedAlbum(ctx, store.InsertSavedAlbumParams{
		ListenerID:    listener.ID,
		Title:         "Beta",
		PrimaryArtist: &artist,
		ProviderName:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = st.DeleteListener(ctx, listener.ID)
	})

	payloadNew, err := json.Marshal(removeDisambigRoot{Kind: removeDisambigKind, Candidates: []removeDisambigItem{
		{ID: idB, Label: "Beta"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.CreateDisambiguationSession(ctx, listener.ID, payloadNew, store.DefaultSessionTTL)
	if err != nil {
		t.Fatal(err)
	}

	lib := &LibraryService{Store: st}
	staleID := uuid.NewString()
	msg, ok, err := lib.TryProcessRemovePick(ctx, store.ListenerSourceTelegram, ext, staleID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected handled with safe copy")
	}
	if msg != removePickKeyedStaleCopy() {
		t.Fatalf("message %q", msg)
	}

	rows, err := st.ListSavedAlbumRowsForListener(ctx, listener.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != idB {
		t.Fatalf("saved rows: %+v", rows)
	}
}

func TestTryProcessRemovePick_textPathIgnoresNonRemoveLatest(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skip Postgres integration test")
	}
	ctx := context.Background()
	migDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RunMigrations(migDir, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := store.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(st.Close)

	ext := "test-rmp-text-" + uuid.NewString()
	listener, err := st.UpsertListener(ctx, store.ListenerSourceTelegram, ext, "", "")
	if err != nil {
		t.Fatal(err)
	}
	artist := "Artist"
	idA, err := st.InsertSavedAlbum(ctx, store.InsertSavedAlbumParams{
		ListenerID:    listener.ID,
		Title:         "Alpha",
		PrimaryArtist: &artist,
		ProviderName:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = st.DeleteListener(ctx, listener.ID)
	})

	removeFirst, err := json.Marshal(removeDisambigRoot{Kind: removeDisambigKind, Candidates: []removeDisambigItem{
		{ID: idA, Label: "Alpha"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateDisambiguationSession(ctx, listener.ID, removeFirst, store.DefaultSessionTTL); err != nil {
		t.Fatal(err)
	}
	albumDisambig, err := json.Marshal([]AlbumCandidate{{Title: "Other Album", PrimaryArtist: "X", Provider: "t", ProviderRef: "r"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateDisambiguationSession(ctx, listener.ID, albumDisambig, store.DefaultSessionTTL); err != nil {
		t.Fatal(err)
	}

	lib := &LibraryService{Store: st}
	_, ok, err := lib.TryProcessRemovePick(ctx, store.ListenerSourceTelegram, ext, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected not handled when latest session is album disambiguation")
	}

	rows, err := st.ListSavedAlbumRowsForListener(ctx, listener.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows: %+v", rows)
	}
}

func TestTryProcessRemovePick_textPathLatestRemoveSaved(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skip Postgres integration test")
	}
	ctx := context.Background()
	migDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RunMigrations(migDir, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := store.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(st.Close)

	ext := "test-rmp-text2-" + uuid.NewString()
	listener, err := st.UpsertListener(ctx, store.ListenerSourceTelegram, ext, "", "")
	if err != nil {
		t.Fatal(err)
	}
	artist := "Artist"
	idA, err := st.InsertSavedAlbum(ctx, store.InsertSavedAlbumParams{
		ListenerID:    listener.ID,
		Title:         "Alpha",
		PrimaryArtist: &artist,
		ProviderName:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = st.DeleteListener(ctx, listener.ID)
	})

	removePayload, err := json.Marshal(removeDisambigRoot{Kind: removeDisambigKind, Candidates: []removeDisambigItem{
		{ID: idA, Label: "Alpha"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateDisambiguationSession(ctx, listener.ID, removePayload, store.DefaultSessionTTL); err != nil {
		t.Fatal(err)
	}

	lib := &LibraryService{Store: st}
	msg, ok, err := lib.TryProcessRemovePick(ctx, store.ListenerSourceTelegram, ext, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || msg == "" {
		t.Fatalf("handled=%v msg=%q", ok, msg)
	}
	rows, err := st.ListSavedAlbumRowsForListener(ctx, listener.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected album removed: %+v", rows)
	}
}
