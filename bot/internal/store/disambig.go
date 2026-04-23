package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreateDisambiguationSession stores candidates (JSON) for a listener pick flow.
func (s *Store) CreateDisambiguationSession(ctx context.Context, listenerID string, candidatesJSON []byte, ttl time.Duration) (string, error) {
	var id string
	expires := time.Now().Add(ttl)
	err := s.pool.QueryRow(ctx, `
		INSERT INTO disambiguation_sessions (listener_id, candidates, created_at, expires_at)
		VALUES ($1, $2, now(), $3)
		RETURNING id::text
	`, listenerID, candidatesJSON, expires).Scan(&id)
	return id, err
}

// LatestOpenDisambiguationSession returns the newest non-expired session for a platform user, if any.
func (s *Store) LatestOpenDisambiguationSession(ctx context.Context, source, externalID string) (*Session, []byte, error) {
	var sid string
	var raw []byte
	err := s.pool.QueryRow(ctx, `
		SELECT s.id, s.candidates
		FROM disambiguation_sessions s
		JOIN listeners l ON l.id = s.listener_id
		WHERE l.source = $1 AND l.external_id = $2 AND s.expires_at > now()
		ORDER BY s.created_at DESC
		LIMIT 1
	`, source, externalID).Scan(&sid, &raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return &Session{ID: sid}, raw, nil
}

// DeleteDisambiguationSession removes a session after a successful pick.
func (s *Store) DeleteDisambiguationSession(ctx context.Context, sessionID string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM disambiguation_sessions WHERE id = $1`, sessionID)
	if err != nil {
		return err
	}
	_ = ct
	return nil
}

// DeleteDisambigForListener clears pending picks before creating a new search session.
func (s *Store) DeleteDisambigForListener(ctx context.Context, listenerID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM disambiguation_sessions WHERE listener_id = $1::uuid`, listenerID)
	return err
}
