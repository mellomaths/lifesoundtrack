package core

import "context"

// MetadataOrchestrator is the port for metadata search (chain behind the scenes).
type MetadataOrchestrator interface {
	// Search returns 0..N candidates, ordered by relevance; ErrNoMatch from metadata means no result.
	Search(ctx context.Context, query string) ([]AlbumCandidate, error)
}
