package store

import (
	"context"
)

// DeleteSavedAlbumForListener deletes one saved row by id when it belongs to listenerID.
// Returns true if a row was deleted.
func (s *Store) DeleteSavedAlbumForListener(ctx context.Context, albumID, listenerID string) (bool, error) {
	if albumID == "" || listenerID == "" {
		return false, nil
	}
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM saved_albums WHERE id = $1::uuid AND listener_id = $2::uuid
	`, albumID, listenerID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
