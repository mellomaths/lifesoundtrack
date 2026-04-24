package store

import (
	"os"
	"strings"
	"testing"
)

// TestMigrationsEnforceListenerIdentity documents UNIQUE (source, external_id) for US3 (tasks T029).
func TestMigrationsEnforceListenerIdentity(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("../../migrations/00001_init_listeners_saved_albums_disambig.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, "listeners_source_external") && !strings.Contains(s, "UNIQUE (source, external_id)") {
		t.Fatal("expected listeners UNIQUE (source, external_id) in initial migration")
	}
}
