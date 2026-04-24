package metadata

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
	"github.com/sony/gobreaker"
)

// Chain implements [core.MetadataOrchestrator] with
// Spotify → iTunes → Last.fm → MusicBrainz (per [spec FR-002] and contracts).
type Chain struct {
	http *http.Client
	// MusicBrainzUserAgent is required when MusicBrainz is enabled (contact per MB policy).
	MusicBrainzUserAgent string
	lastfmKey            string
	spotifyClientID      string
	spotifyClientSecret  string
	enableSpotify          bool
	enableITunes           bool
	enableLastfm           bool
	enableMusicBrainz      bool
	log                    *slog.Logger
	missingSpotifyCreds    sync.Once
	spBrk, itBrk, lfBrk, mbBrk *gobreaker.CircuitBreaker
	mbThrottle *mbThrottle
	// Spotify access token (in-memory; spotifyAccessToken in spotify.go)
	spotifyTokMu   sync.Mutex
	spotifyToken   string
	spotifyTokenExp time.Time
}

var _ core.MetadataOrchestrator = (*Chain)(nil)

// ChainConfig wires environment-driven metadata behavior (default: all flags true).
type ChainConfig struct {
	HTTP                 *http.Client
	LastfmAPIKey         string
	MusicBrainzUserAgent string
	SpotifyClientID      string
	SpotifyClientSecret  string
	EnableSpotify        bool
	EnableITunes         bool
	EnableLastfm         bool
	EnableMusicBrainz    bool
	Log                  *slog.Logger
}

// NewChain builds the provider stack. If cfg.HTTP is nil, a default client (12s timeout) is used.
func NewChain(cfg ChainConfig) *Chain {
	h := cfg.HTTP
	if h == nil {
		h = &http.Client{Timeout: 12 * time.Second}
	}
	ua := cfg.MusicBrainzUserAgent
	if ua == "" {
		ua = "LifeSoundTrackBot/1.0 (https://github.com/mellomaths/lifesoundtrack)"
	}
	return &Chain{
		http:                 h,
		MusicBrainzUserAgent: ua,
		lastfmKey:         cfg.LastfmAPIKey,
		spotifyClientID:   cfg.SpotifyClientID,
		spotifyClientSecret: cfg.SpotifyClientSecret,
		enableSpotify:     cfg.EnableSpotify,
		enableITunes:      cfg.EnableITunes,
		enableLastfm:      cfg.EnableLastfm,
		enableMusicBrainz: cfg.EnableMusicBrainz,
		log:                  cfg.Log,
		spBrk:                newProviderBreaker("spotify"),
		itBrk:                newProviderBreaker("itunes"),
		lfBrk:                newProviderBreaker("lastfm"),
		mbBrk:                newProviderBreaker("musicbrainz"),
		mbThrottle:           newMBThrottle(),
	}
}

func newProviderBreaker(name string) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,
		Interval:    0,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(c gobreaker.Counts) bool { return c.ConsecutiveFailures >= 3 },
	})
}

// Search implements [core.MetadataOrchestrator]. Result: relevance order, cap 2 in UI.
func (c *Chain) Search(ctx context.Context, query string) ([]core.AlbumCandidate, error) {
	if c == nil {
		return nil, fmt.Errorf("nil chain")
	}
	if query == "" {
		return nil, core.ErrNoMatch
	}
	if !c.enableSpotify && !c.enableITunes && !c.enableLastfm && !c.enableMusicBrainz {
		return nil, core.ErrAllProvidersExhausted
	}
	if c.enableSpotify && (c.spotifyClientID == "" || c.spotifyClientSecret == "") {
		c.missingSpotifyCreds.Do(func() {
			if c.log != nil {
				c.log.Warn("SPOTIFY_CLIENT_ID/SPOTIFY_CLIENT_SECRET missing; skipping Spotify in metadata chain",
					"component", "metadata")
			}
		})
	}
	var eSp, eIt, eLf, eMb error
	sp, err := c.runSpotifySearch(ctx, query)
	eSp = err
	if err == nil && len(sp) > 0 {
		return capTop2(sp), nil
	}
	it, err := c.runITunesSearch(ctx, query)
	eIt = err
	if err == nil && len(it) > 0 {
		return capTop2(it), nil
	}
	lf, err := c.runLastfmSearch(ctx, query)
	eLf = err
	if err == nil && len(lf) > 0 {
		return capTop2(lf), nil
	}
	mb, err := c.runMusicBrainzSearch(ctx, query)
	eMb = err
	if err == nil && len(mb) > 0 {
		return capTop2(mb), nil
	}
	if c.allEnabledRingsExhausted(eSp, eIt, eLf, eMb) {
		return nil, core.ErrAllProvidersExhausted
	}
	return nil, core.ErrNoMatch
}

// allEnabledRingsExhausted is true when every *enabled* ring is in an unavailable
// (breaker open or missing key/creds) state—aligned with gobreaker open semantics in legacy orchestrator.
func (c *Chain) allEnabledRingsExhausted(eSp, eIt, eLf, eMb error) bool {
	if c.enableSpotify && !c.ringSpotifyUnusable(eSp) {
		return false
	}
	if c.enableITunes && !c.ringITunesUnusable(eIt) {
		return false
	}
	if c.enableLastfm && !c.ringLastfmUnusable(eLf) {
		return false
	}
	if c.enableMusicBrainz && !c.ringMusicBrainzUnusable(eMb) {
		return false
	}
	return true
}

// ring*Unusable means the ring cannot serve requests (or is disabled, ignored by caller).
func (c *Chain) ringSpotifyUnusable(err error) bool {
	if !c.enableSpotify {
		return true
	}
	if c.spotifyClientID == "" || c.spotifyClientSecret == "" {
		return true
	}
	return errors.Is(err, gobreaker.ErrOpenState)
}

func (c *Chain) ringITunesUnusable(err error) bool {
	if !c.enableITunes {
		return true
	}
	return errors.Is(err, gobreaker.ErrOpenState)
}

func (c *Chain) ringLastfmUnusable(err error) bool {
	if !c.enableLastfm {
		return true
	}
	if c.lastfmKey == "" {
		return true
	}
	return errors.Is(err, gobreaker.ErrOpenState)
}

func (c *Chain) ringMusicBrainzUnusable(err error) bool {
	if !c.enableMusicBrainz {
		return true
	}
	return errors.Is(err, gobreaker.ErrOpenState)
}

func capTop2(cands []core.AlbumCandidate) []core.AlbumCandidate {
	if len(cands) <= 2 {
		return cands
	}
	return cands[:2]
}
