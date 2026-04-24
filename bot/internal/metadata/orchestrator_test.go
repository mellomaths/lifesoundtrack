package metadata

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
)

func TestSearch_AllMetadataFlagsOff(t *testing.T) {
	c := NewChain(ChainConfig{
		HTTP:                 &http.Client{},
		EnableSpotify:        false,
		EnableITunes:         false,
		EnableLastfm:         false,
		EnableMusicBrainz:    false,
	})
	_, err := c.Search(context.Background(), "abbey road")
	if !errors.Is(err, core.ErrAllProvidersExhausted) {
		t.Fatalf("expected ErrAllProvidersExhausted, got %v", err)
	}
}

func TestSearch_EmptyQuery_NoMatch(t *testing.T) {
	c := NewChain(ChainConfig{HTTP: &http.Client{}})
	_, err := c.Search(context.Background(), "")
	if !errors.Is(err, core.ErrNoMatch) {
		t.Fatalf("expected ErrNoMatch, got %v", err)
	}
}
