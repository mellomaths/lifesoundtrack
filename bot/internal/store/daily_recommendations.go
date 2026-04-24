package store

import (
	"context"
)

// ListenerSourceTelegram matches the Telegram adapter's listener source value ("telegram").
const ListenerSourceTelegram = "telegram"

// ListTelegramListenerIDsWithSavedAlbums returns listener IDs that have at least one saved_albums row,
// for Telegram only, stable-ordered by created_at then id.
//
// It uses EXISTS instead of JOIN + DISTINCT so ORDER BY can reference listeners columns without
// PostgreSQL rejecting the statement (SQLSTATE 42P10: DISTINCT + ORDER BY mismatch).
func (s *Store) ListTelegramListenerIDsWithSavedAlbums(ctx context.Context) ([]string, error) {
	targets, err := s.ListTelegramDailyTargets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(targets))
	for i, t := range targets {
		out[i] = t.ListenerID
	}
	return out, nil
}
