package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// DailyListenerTarget is a Telegram listener eligible for the daily job (has saved albums).
type DailyListenerTarget struct {
	ListenerID string
	ExternalID string
}

// ListTelegramDailyTargets returns Telegram listeners that have at least one saved album,
// ordered by listener created_at (stable). See FR-013–FR-015.
func (s *Store) ListTelegramDailyTargets(ctx context.Context) ([]DailyListenerTarget, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT l.id::text, l.external_id
		FROM listeners l
		WHERE l.source = $1
		  AND EXISTS (SELECT 1 FROM saved_albums s WHERE s.listener_id = l.id)
		ORDER BY l.created_at ASC, l.id::text ASC
	`, ListenerSourceTelegram)
	if err != nil {
		return nil, fmt.Errorf("list telegram daily targets: %w", err)
	}
	defer rows.Close()

	var out []DailyListenerTarget
	for rows.Next() {
		var t DailyListenerTarget
		if err := rows.Scan(&t.ListenerID, &t.ExternalID); err != nil {
			return nil, fmt.Errorf("scan daily target: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// SavedAlbumForDaily is a saved_albums row used for fair rotation and messaging.
type SavedAlbumForDaily struct {
	ID                 string
	Title              string
	PrimaryArtist      *string
	Year               *int
	ProviderName       string
	ProviderAlbumID    *string
	ArtURL             *string
	Extra              []byte
	LastRecommendedAt *time.Time
}

// ListSavedAlbumsForDaily loads all saved albums for a listener (rotation input).
func (s *Store) ListSavedAlbumsForDaily(ctx context.Context, listenerID string) ([]SavedAlbumForDaily, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, title, primary_artist, year, provider_name, provider_album_id, art_url, extra, last_recommended_at
		FROM saved_albums
		WHERE listener_id = $1::uuid
	`, listenerID)
	if err != nil {
		return nil, fmt.Errorf("list saved albums for daily: %w", err)
	}
	defer rows.Close()

	var out []SavedAlbumForDaily
	for rows.Next() {
		var r SavedAlbumForDaily
		var artist sql.NullString
		var year sql.NullInt32
		var provAlbum sql.NullString
		var art sql.NullString
		var last sql.NullTime
		if err := rows.Scan(
			&r.ID, &r.Title, &artist, &year, &r.ProviderName, &provAlbum, &art, &r.Extra, &last,
		); err != nil {
			return nil, fmt.Errorf("scan saved album: %w", err)
		}
		r.PrimaryArtist = nullStringPtr(artist)
		if year.Valid {
			y := int(year.Int32)
			r.Year = &y
		}
		r.ProviderAlbumID = nullStringPtr(provAlbum)
		r.ArtURL = nullStringPtr(art)
		if last.Valid {
			t := last.Time
			r.LastRecommendedAt = &t
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func nullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

// RecordRecommendationParams is input for RecordRecommendationTx (FR-007).
type RecordRecommendationParams struct {
	RunID              string
	ListenerID         string
	SavedAlbumID       string
	TitleSnapshot      string
	ArtistSnapshot     *string
	YearSnapshot       *int
	SpotifyURLSnapshot *string
	SentAt             time.Time
}

// RecordRecommendationTx updates last_recommended_at and inserts recommendations in one transaction.
func (s *Store) RecordRecommendationTx(ctx context.Context, p RecordRecommendationParams) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE saved_albums SET last_recommended_at = $1 WHERE id = $2::uuid
	`, p.SentAt, p.SavedAlbumID); err != nil {
		return fmt.Errorf("update last_recommended_at: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO recommendations (
			run_id, listener_id, saved_album_id,
			title_snapshot, artist_snapshot, year_snapshot, spotify_url_snapshot, sent_at
		) VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8)
	`, p.RunID, p.ListenerID, p.SavedAlbumID,
		p.TitleSnapshot, p.ArtistSnapshot, p.YearSnapshot, p.SpotifyURLSnapshot, p.SentAt,
	); err != nil {
		return fmt.Errorf("insert recommendation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
