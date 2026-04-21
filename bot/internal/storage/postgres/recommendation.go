package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/mellomaths/lifesoundtrack/bot/internal/ports"
)

// ListTelegramRecipientsWithAlbums returns internal users who have albums and a Telegram identity.
func (s *Store) ListTelegramRecipientsWithAlbums(ctx context.Context) ([]ports.TelegramRecipient, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ui.user_id, ui.external_id
		FROM album_interests ai
		INNER JOIN user_identities ui ON ui.user_id = ai.user_id AND ui.source = $1
	`, ports.SourceTelegram)
	if err != nil {
		return nil, fmt.Errorf("list recipients: %w", err)
	}
	defer rows.Close()

	var out []ports.TelegramRecipient
	for rows.Next() {
		var userID int64
		var ext string
		if err := rows.Scan(&userID, &ext); err != nil {
			return nil, fmt.Errorf("scan recipient: %w", err)
		}
		chatID, err := strconv.ParseInt(ext, 10, 64)
		if err != nil {
			continue
		}
		out = append(out, ports.TelegramRecipient{
			InternalUserID: userID,
			TelegramChatID: chatID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipients: %w", err)
	}
	return out, nil
}

// PickNextRecommendation chooses one album using fair tier + random tie-break (spec: daily recommendation).
func (s *Store) PickNextRecommendation(ctx context.Context, userID int64) (*ports.PickedRecommendation, error) {
	row := s.pool.QueryRow(ctx, `
		WITH has_null AS (
			SELECT EXISTS (
				SELECT 1 FROM album_interests WHERE user_id = $1 AND last_recommended_at IS NULL
			) AS v
		)
		SELECT ai.id, ai.user_id, ai.album_title, ai.artist
		FROM album_interests ai, has_null hn
		WHERE ai.user_id = $1
		AND (
			(hn.v AND ai.last_recommended_at IS NULL)
			OR (
				NOT hn.v
				AND ai.last_recommended_at = (
					SELECT MIN(ai2.last_recommended_at)
					FROM album_interests ai2
					WHERE ai2.user_id = $1
				)
			)
		)
		ORDER BY random()
		LIMIT 1
	`, userID)

	var id, uid int64
	var title, artist string
	if err := row.Scan(&id, &uid, &title, &artist); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("pick recommendation: %w", err)
	}
	return &ports.PickedRecommendation{
		AlbumInterestID: id,
		UserID:          uid,
		AlbumTitle:      title,
		Artist:          artist,
	}, nil
}

// RecordRecommendation updates last_recommended_at and inserts recommendation_audit in one transaction.
func (s *Store) RecordRecommendation(ctx context.Context, in ports.RecordRecommendationInput) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx,
		`UPDATE album_interests SET last_recommended_at = now() WHERE id = $1 AND user_id = $2`,
		in.AlbumInterestID, in.UserID,
	)
	if err != nil {
		return fmt.Errorf("update last_recommended_at: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("no album_interest row updated for id=%d user=%d", in.AlbumInterestID, in.UserID)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO recommendation_audit (album_interest_id, user_id, album_title, artist, recommended_at)
		 VALUES ($1, $2, $3, $4, now())`,
		in.AlbumInterestID, in.UserID, in.AlbumTitle, in.Artist,
	)
	if err != nil {
		return fmt.Errorf("insert recommendation_audit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

var _ ports.RecommendationStore = (*Store)(nil)
