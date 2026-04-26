package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// AlbumListSessionRow is one row from album_list_sessions.
type AlbumListSessionRow struct {
	ID               string
	ListenerID       string
	ArtistFilterNorm *string
	CurrentPage      int
	ExpiresAt        time.Time
}

// SavedAlbumListRow is one saved album row for /list display.
type SavedAlbumListRow struct {
	ID            string
	Title         string
	PrimaryArtist *string
	Year          *int
}

// ListenerIDBySourceExternal returns the listener UUID or an empty string if no row exists.
func (s *Store) ListenerIDBySourceExternal(ctx context.Context, source, externalID string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text FROM listeners WHERE source = $1 AND external_id = $2
	`, source, externalID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return id, nil
}

// CountSavedAlbumsForListener counts matching saves; artistNorm is a lowercase substring needle, or nil for all albums.
func (s *Store) CountSavedAlbumsForListener(ctx context.Context, listenerID string, artistNorm *string) (int64, error) {
	if listenerID == "" {
		return 0, nil
	}
	var n int64
	var err error
	if artistNorm == nil {
		err = s.pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM saved_albums WHERE listener_id = $1::uuid
		`, listenerID).Scan(&n)
	} else {
		err = s.pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM saved_albums
			WHERE listener_id = $1::uuid
			  AND strpos(lower(coalesce(primary_artist, '')), $2) > 0
		`, listenerID, *artistNorm).Scan(&n)
	}
	if err != nil {
		return 0, err
	}
	return n, nil
}

// ListSavedAlbumsPage returns one page ordered by created_at DESC, id DESC.
func (s *Store) ListSavedAlbumsPage(ctx context.Context, listenerID string, artistNorm *string, offset, limit int) ([]SavedAlbumListRow, error) {
	if listenerID == "" {
		return nil, nil
	}
	var rows pgx.Rows
	var err error
	if artistNorm == nil {
		rows, err = s.pool.Query(ctx, `
			SELECT id::text, title, primary_artist, year
			FROM saved_albums
			WHERE listener_id = $1::uuid
			ORDER BY created_at DESC, id DESC
			LIMIT $2 OFFSET $3
		`, listenerID, limit, offset)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id::text, title, primary_artist, year
			FROM saved_albums
			WHERE listener_id = $1::uuid
			  AND strpos(lower(coalesce(primary_artist, '')), $2) > 0
			ORDER BY created_at DESC, id DESC
			LIMIT $3 OFFSET $4
		`, listenerID, *artistNorm, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SavedAlbumListRow, 0, limit)
	for rows.Next() {
		var r SavedAlbumListRow
		if err := rows.Scan(&r.ID, &r.Title, &r.PrimaryArtist, &r.Year); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// InsertAlbumListSession creates a paging session row (multi-page lists only).
func (s *Store) InsertAlbumListSession(ctx context.Context, listenerID string, artistFilterNorm *string, currentPage int, ttl time.Duration) (sessionID string, err error) {
	if listenerID == "" {
		return "", fmt.Errorf("empty listener id for list session")
	}
	expires := time.Now().Add(ttl)
	err = s.pool.QueryRow(ctx, `
		INSERT INTO album_list_sessions (listener_id, artist_filter_norm, current_page, expires_at)
		VALUES ($1::uuid, $2, $3, $4)
		RETURNING id::text
	`, listenerID, artistFilterNorm, currentPage, expires).Scan(&sessionID)
	return sessionID, err
}

// GetAlbumListSession loads a session by id (any expiry).
func (s *Store) GetAlbumListSession(ctx context.Context, sessionID string) (*AlbumListSessionRow, error) {
	var r AlbumListSessionRow
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, listener_id::text, artist_filter_norm, current_page, expires_at
		FROM album_list_sessions
		WHERE id = $1::uuid
	`, sessionID).Scan(&r.ID, &r.ListenerID, &r.ArtistFilterNorm, &r.CurrentPage, &r.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

// LatestOpenAlbumListSession returns the newest non-expired session for a listener, if any.
func (s *Store) LatestOpenAlbumListSession(ctx context.Context, listenerID string) (*AlbumListSessionRow, error) {
	if listenerID == "" {
		return nil, nil
	}
	var r AlbumListSessionRow
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, listener_id::text, artist_filter_norm, current_page, expires_at
		FROM album_list_sessions
		WHERE listener_id = $1::uuid AND expires_at > now()
		ORDER BY created_at DESC
		LIMIT 1
	`, listenerID).Scan(&r.ID, &r.ListenerID, &r.ArtistFilterNorm, &r.CurrentPage, &r.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

// ListSavedAlbumRowsForListener returns all saved rows for a listener (for /remove in-Go title matching).
func (s *Store) ListSavedAlbumRowsForListener(ctx context.Context, listenerID string) ([]SavedAlbumListRow, error) {
	if listenerID == "" {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, title, primary_artist, year
		FROM saved_albums
		WHERE listener_id = $1::uuid
		ORDER BY created_at ASC, id ASC
	`, listenerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SavedAlbumListRow
	for rows.Next() {
		var r SavedAlbumListRow
		if err := rows.Scan(&r.ID, &r.Title, &r.PrimaryArtist, &r.Year); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateAlbumListSessionPage updates the cursor page for text /list next|back and callback navigation.
func (s *Store) UpdateAlbumListSessionPage(ctx context.Context, sessionID string, page int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE album_list_sessions SET current_page = $2 WHERE id = $1::uuid
	`, sessionID, page)
	return err
}
