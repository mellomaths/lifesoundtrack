package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestOpenDisambiguationSessionForListener(t *testing.T) {
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

	extOwn := "test-disambig-own-" + uuid.NewString()
	extOther := "test-disambig-other-" + uuid.NewString()
	listenerOwn, err := st.UpsertListener(ctx, ListenerSourceTelegram, extOwn, "", "")
	if err != nil {
		t.Fatal(err)
	}
	listenerOther, err := st.UpsertListener(ctx, ListenerSourceTelegram, extOther, "", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, id := range []string{listenerOwn.ID, listenerOther.ID} {
			_, _ = st.pool.Exec(ctx, `DELETE FROM disambiguation_sessions WHERE listener_id = $1::uuid`, id)
			_, _ = st.pool.Exec(ctx, `DELETE FROM listeners WHERE id = $1::uuid`, id)
		}
	})

	payload, err := json.Marshal(map[string]any{"kind": "remove_saved", "candidates": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	sid, err := st.CreateDisambiguationSession(ctx, listenerOwn.ID, payload, DefaultSessionTTL)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid", func(t *testing.T) {
		raw, err := st.OpenDisambiguationSessionForListener(ctx, sid, listenerOwn.ID)
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) != string(payload) {
			t.Fatalf("candidates: got %s want %s", raw, payload)
		}
	})

	t.Run("wrong_listener", func(t *testing.T) {
		_, err := st.OpenDisambiguationSessionForListener(ctx, sid, listenerOther.ID)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("want pgx.ErrNoRows, got %v", err)
		}
	})

	t.Run("wrong_session_id", func(t *testing.T) {
		_, err := st.OpenDisambiguationSessionForListener(ctx, uuid.NewString(), listenerOwn.ID)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("want pgx.ErrNoRows, got %v", err)
		}
	})

	t.Run("expired", func(t *testing.T) {
		sidExp, err := st.CreateDisambiguationSession(ctx, listenerOwn.ID, payload, -time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		_, err = st.OpenDisambiguationSessionForListener(ctx, sidExp, listenerOwn.ID)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("want pgx.ErrNoRows, got %v", err)
		}
	})
}
