package metadata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
	"github.com/sony/gobreaker"
)

// Chain implements [core.MetadataOrchestrator] with MusicBrainz → Last.fm → iTunes.
type Chain struct {
	http *http.Client
	// MusicBrainzUserAgent is required (contact string per MusicBrainz policy).
	MusicBrainzUserAgent string
	lastfmKey            string
	mbBrk                *gobreaker.CircuitBreaker
	lfBrk                *gobreaker.CircuitBreaker
	itBrk                *gobreaker.CircuitBreaker
	mbThrottle           *mbThrottle
}

var _ core.MetadataOrchestrator = (*Chain)(nil)

// NewChain builds the default provider stack. If httpClient is nil, a reasonable default is used.
func NewChain(httpClient *http.Client, lastfmAPIKey, musicBrainzUserAgent string) *Chain {
	h := httpClient
	if h == nil {
		h = &http.Client{Timeout: 12 * time.Second}
	}
	return &Chain{
		http:                 h,
		MusicBrainzUserAgent: musicBrainzUserAgent,
		lastfmKey:            lastfmAPIKey,
		mbBrk:                newProviderBreaker("musicbrainz"),
		lfBrk:                newProviderBreaker("lastfm"),
		itBrk:                newProviderBreaker("itunes"),
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

// Search implements [core.MetadataOrchestrator]. Result order: best relevance first, max 3.
func (c *Chain) Search(ctx context.Context, query string) ([]core.AlbumCandidate, error) {
	if c == nil {
		return nil, fmt.Errorf("nil chain")
	}
	if query == "" {
		return nil, core.ErrNoMatch
	}
	mb, err := c.runMusicBrainzSearch(ctx, query)
	if err == nil && len(mb) > 0 {
		return capTop3(mb), nil
	}
	lf, err2 := c.runLastfmSearch(ctx, query)
	if err2 == nil && len(lf) > 0 {
		return capTop3(lf), nil
	}
	it, err3 := c.runITunesSearch(ctx, query)
	if err3 == nil && len(it) > 0 {
		return capTop3(it), nil
	}
	// All rings returned nothing; only treat as "provider_exhausted" if each invoked ring is open.
	lfOpen := c.lastfmKey == "" || errors.Is(err2, gobreaker.ErrOpenState)
	if errors.Is(err, gobreaker.ErrOpenState) && lfOpen && errors.Is(err3, gobreaker.ErrOpenState) {
		return nil, core.ErrAllProvidersExhausted
	}
	_ = err2
	_ = err3
	return nil, core.ErrNoMatch
}

func capTop3(cands []core.AlbumCandidate) []core.AlbumCandidate {
	if len(cands) <= 3 {
		return cands
	}
	return cands[:3]
}
