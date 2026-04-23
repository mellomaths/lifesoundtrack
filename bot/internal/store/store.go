package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultSessionTTL is how long a disambiguation pick stays valid (FR-009).
const DefaultSessionTTL = 15 * time.Minute

// Store is a thin PostgreSQL access layer.
type Store struct {
	pool *pgxpool.Pool
}

// OpenPool dials the database and verifies connectivity.
func OpenPool(ctx context.Context, databaseURL string) (*Store, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("empty database url")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() {
	if s == nil || s.pool == nil {
		return
	}
	s.pool.Close()
}

// Listener is a row in listeners.
type Listener struct {
	ID string
}

// Session identifies an open disambiguation session.
type Session struct {
	ID string
}
