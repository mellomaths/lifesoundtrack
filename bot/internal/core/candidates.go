package core

// AlbumCandidate is a normalized result from a metadata provider (see contracts).
type AlbumCandidate struct {
	Title         string
	PrimaryArtist string
	Year          *int
	Genres        []string
	Relevance     float64
	Provider      string
	ProviderRef   string
	ArtURL        string
}

// CapGenres enforces a max of 8 genre labels for storage and payloads.
func CapGenres(g []string) []string {
	if len(g) <= 8 {
		return g
	}
	return g[:8]
}
