package metadata

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
)

var spotifyRedirectAllowHosts = map[string]struct{}{
	"spoti.fi":             {},
	"www.spotify.com":      {},
	"open.spotify.com":     {},
	"accounts.spotify.com": {},
}

const spotifyShareResolveTimeout = 4 * time.Second

// spotifyShareRedirectPolicy enforces HTTPS, allowlisted hosts, and a redirect hop cap (shared with tests).
func spotifyShareRedirectPolicy() func(*http.Request, []*http.Request) error {
	var hops int
	return func(req *http.Request, via []*http.Request) error {
		hops++
		if hops > 5 {
			return fmt.Errorf("too many redirects")
		}
		if req.URL.Scheme != "https" {
			return fmt.Errorf("non-https redirect")
		}
		h := strings.ToLower(req.URL.Hostname())
		if _, ok := spotifyRedirectAllowHosts[h]; !ok {
			return fmt.Errorf("redirect host not allowed")
		}
		return nil
	}
}

func newSpotifyShareHTTPClient() *http.Client {
	return &http.Client{
		Timeout:       spotifyShareResolveTimeout,
		CheckRedirect: spotifyShareRedirectPolicy(),
	}
}

// resolveSpotifyShareToAlbumID follows HTTPS redirects with an allowlist and hop cap, then extracts an open.spotify.com album id.
func resolveSpotifyShareToAlbumID(ctx context.Context, shareURL string) (albumID string, err error) {
	return resolveSpotifyShareWithHTTPClient(ctx, shareURL, newSpotifyShareHTTPClient())
}

// resolveSpotifyShareWithHTTPClient is like resolveSpotifyShareToAlbumID but uses the given client (for tests with a custom http.RoundTripper).
func resolveSpotifyShareWithHTTPClient(ctx context.Context, shareURL string, client *http.Client) (string, error) {
	u0, err := url.Parse(strings.TrimSpace(shareURL))
	if err != nil {
		return "", fmt.Errorf("parse share url: %w", err)
	}
	if u0.Scheme != "https" {
		return "", fmt.Errorf("share url must use https")
	}
	host0 := strings.ToLower(u0.Hostname())
	if host0 != "spoti.fi" && host0 != "www.spotify.com" {
		return "", fmt.Errorf("unsupported share host")
	}
	if client == nil {
		client = newSpotifyShareHTTPClient()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u0.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))

	final := resp.Request.URL
	if final == nil {
		return "", fmt.Errorf("no final url")
	}
	id, ok := core.SpotifyAlbumIDFromParsedURL(final)
	if !ok {
		return "", fmt.Errorf("did not resolve to a spotify album page")
	}
	return id, nil
}
