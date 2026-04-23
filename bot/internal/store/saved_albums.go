package store

import (
	"context"
)

// InsertSavedAlbumParams maps to the saved_albums table.
type InsertSavedAlbumParams struct {
	ListenerID      string
	UserQueryText   *string
	Title           string
	PrimaryArtist   *string
	Year            *int
	Genres          []string
	ProviderName    string
	ProviderAlbumID *string
	ArtURL          *string
	Extra           []byte
}

// InsertSavedAlbum records one user save event.
func (s *Store) InsertSavedAlbum(ctx context.Context, p InsertSavedAlbumParams) (id string, err error) {
	var genres any = p.Genres
	if p.Genres == nil {
		genres = nil
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO saved_albums (
			listener_id, user_query_text, title, primary_artist, year, genres,
			provider_name, provider_album_id, art_url, extra, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
		RETURNING id::text
	`, p.ListenerID, p.UserQueryText, p.Title, p.PrimaryArtist, p.Year, genres,
		p.ProviderName, p.ProviderAlbumID, p.ArtURL, p.Extra,
	).Scan(&id)
	return id, err
}
