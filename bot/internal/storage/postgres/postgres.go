package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mellomaths/lifesoundtrack/bot/internal/ports"
)

//go:embed schema.sql
var schemaSQL string

// Store implements ports.InterestWriter against PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pgx pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// NewPool builds a connection pool from DATABASE_URL.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return pool, nil
}

// Migrate applies embedded DDL (idempotent CREATE IF NOT EXISTS).
func (s *Store) Migrate(ctx context.Context) error {
	if s.pool == nil {
		return fmt.Errorf("migrate: pool is nil")
	}
	if _, err := s.pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

// AddInterest upserts user / identity rows and inserts album_interests in one transaction.
func (s *Store) AddInterest(ctx context.Context, in ports.AddInterestInput) error {
	if in.Source != ports.SourceTelegram {
		return fmt.Errorf("unsupported identity source: %s", in.Source)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID int64
	err = tx.QueryRow(ctx,
		`SELECT user_id FROM user_identities WHERE source = $1 AND external_id = $2`,
		in.Source, in.ExternalID,
	).Scan(&userID)

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("lookup identity: %w", err)
		}

		var nameArg any
		if in.DisplayName != nil {
			nameArg = *in.DisplayName
		}
		err = tx.QueryRow(ctx,
			`INSERT INTO users (name, created_at, updated_at) VALUES ($1, now(), now()) RETURNING id`,
			nameArg,
		).Scan(&userID)
		if err != nil {
			return fmt.Errorf("insert user: %w", err)
		}

		var unameArg any
		if in.Username != nil {
			unameArg = *in.Username
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO user_identities (user_id, source, external_id, username, linked_at) VALUES ($1, $2, $3, $4, now())`,
			userID, in.Source, in.ExternalID, unameArg,
		)
		if err != nil {
			return fmt.Errorf("insert identity: %w", err)
		}
	} else {
		if in.DisplayName != nil {
			_, err = tx.Exec(ctx,
				`UPDATE users SET name = $1, updated_at = now() WHERE id = $2`,
				*in.DisplayName, userID,
			)
			if err != nil {
				return fmt.Errorf("update user name: %w", err)
			}
		}
		if in.Username != nil {
			_, err = tx.Exec(ctx,
				`UPDATE user_identities SET username = $1 WHERE user_id = $2 AND source = $3 AND external_id = $4`,
				*in.Username, userID, in.Source, in.ExternalID,
			)
			if err != nil {
				return fmt.Errorf("update identity username: %w", err)
			}
		}
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO album_interests (user_id, album_title, artist, created_at) VALUES ($1, $2, $3, now())`,
		userID, in.AlbumTitle, in.Artist,
	)
	if err != nil {
		return fmt.Errorf("insert album interest: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

var _ ports.InterestWriter = (*Store)(nil)
