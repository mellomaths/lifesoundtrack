package store

import (
	"context"
)

// UpsertListener ensures one row per (source, external_id) and returns its id.
func (s *Store) UpsertListener(ctx context.Context, source, externalID, displayName, username string) (*Listener, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO listeners (source, external_id, display_name, username, created_at, updated_at)
		VALUES ($1, $2, $3, $4, now(), now())
		ON CONFLICT (source, external_id) DO UPDATE
		SET
			display_name = EXCLUDED.display_name,
			username = EXCLUDED.username,
			updated_at = now()
		RETURNING id::text
	`, source, externalID, nullIfEmpty(displayName), nullIfEmpty(username)).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &Listener{ID: id}, nil
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// DeleteListener removes a listener row; FK cascades delete saved_albums and disambiguation_sessions.
func (s *Store) DeleteListener(ctx context.Context, listenerID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM listeners WHERE id = $1::uuid`, listenerID)
	return err
}
