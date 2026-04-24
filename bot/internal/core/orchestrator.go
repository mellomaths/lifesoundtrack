package core

import "context"

// MetadataOrchestrator is the port for metadata search (chain behind the scenes).
type MetadataOrchestrator interface {
	// Search returns 0..N candidates, ordered by relevance; ErrNoMatch from metadata means no result.
	Search(ctx context.Context, query string) ([]AlbumCandidate, error)
	// LookupSpotifyAlbumByID returns 0 or 1 candidate for a Spotify album id (direct-link path).
	LookupSpotifyAlbumByID(ctx context.Context, albumID string) ([]AlbumCandidate, error)
	// ResolveSpotifyShareURL resolves a supported share URL to a Spotify album id.
	ResolveSpotifyShareURL(ctx context.Context, shareURL string) (albumID string, err error)
}
