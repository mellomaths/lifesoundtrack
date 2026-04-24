package metadata

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
)

// Spotify Web API: client credentials + search (type=album).
const (
	spotifyTokenURL  = "https://accounts.spotify.com/api/token"
	spotifySearchURL = "https://api.spotify.com/v1/search"
	spotifyAlbumURL  = "https://api.spotify.com/v1/albums/"
)

func (c *Chain) runSpotifySearch(ctx context.Context, q string) ([]core.AlbumCandidate, error) {
	if c == nil {
		return nil, fmt.Errorf("nil chain")
	}
	if !c.enableSpotify {
		return nil, nil
	}
	if c.spotifyClientID == "" || c.spotifyClientSecret == "" {
		return nil, nil
	}
	result, err := c.spBrk.Execute(func() (any, error) {
		tok, err := c.spotifyAccessToken(ctx)
		if err != nil {
			return nil, err
		}
		u, err := url.Parse(spotifySearchURL)
		if err != nil {
			return nil, err
		}
		qs := u.Query()
		qs.Set("q", q)
		qs.Set("type", "album")
		qs.Set("limit", "20")
		u.RawQuery = qs.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Accept", "application/json")
		// No custom User-Agent: Spotify may reject; use default from transport.

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			// 429: fall through for breaker / retry in future; for now return error to trip open on repeated failures
			ra := resp.Header.Get("Retry-After")
			return nil, fmt.Errorf("spotify http 429: retry_after=%q", ra)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("spotify http %d", resp.StatusCode)
		}
		return parseSpotifySearchJSON(body)
	})
	if err != nil {
		return nil, err
	}
	return result.([]core.AlbumCandidate), nil
}

type spotifyTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (c *Chain) spotifyAccessToken(ctx context.Context) (string, error) {
	c.spotifyTokMu.Lock()
	defer c.spotifyTokMu.Unlock()
	if c.spotifyToken != "" && time.Now().Before(c.spotifyTokenExp) {
		return c.spotifyToken, nil
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, spotifyTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	basic := base64.StdEncoding.EncodeToString([]byte(c.spotifyClientID + ":" + c.spotifyClientSecret))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("spotify token http %d", resp.StatusCode)
	}
	var tr spotifyTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", err
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("empty spotify access_token")
	}
	c.spotifyToken = tr.AccessToken
	ttl := time.Duration(tr.ExpiresIn) * time.Second
	if ttl < time.Minute {
		ttl = time.Minute
	}
	// refresh one minute before expiry
	c.spotifyTokenExp = time.Now().Add(ttl - 60*time.Second)
	if c.spotifyTokenExp.Before(time.Now()) {
		c.spotifyTokenExp = time.Now().Add(30 * time.Second)
	}
	return c.spotifyToken, nil
}

// spotifySearchJSON is a minimal subset of Spotify search (album) response.
type spotifySearchJSON struct {
	Albums struct {
		Items []struct {
			ID          string                  `json:"id"`
			Name        string                  `json:"name"`
			Artists     []struct{ Name string } `json:"artists"`
			ReleaseDate string                  `json:"release_date"`
			Genres      []string                `json:"genres"`
			Images      []struct{ URL string }  `json:"images"`
		} `json:"items"`
	} `json:"albums"`
}

func parseSpotifySearchJSON(body []byte) ([]core.AlbumCandidate, error) {
	var raw spotifySearchJSON
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if len(raw.Albums.Items) == 0 {
		return nil, nil
	}
	var out []core.AlbumCandidate
	for i, it := range raw.Albums.Items {
		if strings.TrimSpace(it.Name) == "" {
			continue
		}
		var primary string
		if len(it.Artists) > 0 {
			primary = it.Artists[0].Name
		}
		year := yearFromDate(it.ReleaseDate)
		var y *int
		if year != 0 {
			y = intPtr(year)
		}
		rel := 1.0 - float64(i)*0.05
		if rel < 0.1 {
			rel = 0.1
		}
		art := ""
		if len(it.Images) > 0 {
			art = it.Images[0].URL
		}
		out = append(out, core.AlbumCandidate{
			Title:         it.Name,
			PrimaryArtist: primary,
			Year:          y,
			Genres:        core.CapGenres(append([]string{}, it.Genres...)),
			Relevance:     rel,
			Provider:      "spotify",
			ProviderRef:   it.ID,
			ArtURL:        art,
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func yearFromDate(s string) int {
	s = strings.TrimSpace(s)
	if len(s) >= 4 {
		var y int
		_, _ = fmt.Sscanf(s[:4], "%d", &y)
		return y
	}
	return 0
}

func intPtr(n int) *int { return &n }

// spotifyAlbumDetailJSON is a minimal subset of GET /v1/albums/{id}.
type spotifyAlbumDetailJSON struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Artists     []struct{ Name string } `json:"artists"`
	ReleaseDate string                  `json:"release_date"`
	Genres      []string                `json:"genres"`
	Images      []struct{ URL string }  `json:"images"`
}

func parseSpotifyAlbumDetailJSON(body []byte) ([]core.AlbumCandidate, error) {
	var raw spotifyAlbumDetailJSON
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw.ID) == "" || strings.TrimSpace(raw.Name) == "" {
		return nil, nil
	}
	var primary string
	if len(raw.Artists) > 0 {
		primary = raw.Artists[0].Name
	}
	year := yearFromDate(raw.ReleaseDate)
	var y *int
	if year != 0 {
		y = intPtr(year)
	}
	art := ""
	if len(raw.Images) > 0 {
		art = raw.Images[0].URL
	}
	return []core.AlbumCandidate{{
		Title:         raw.Name,
		PrimaryArtist: primary,
		Year:          y,
		Genres:        core.CapGenres(append([]string{}, raw.Genres...)),
		Relevance:     1,
		Provider:      "spotify",
		ProviderRef:   raw.ID,
		ArtURL:        art,
	}}, nil
}

// LookupSpotifyAlbumByID implements [core.MetadataOrchestrator] for the direct-link path.
func (c *Chain) LookupSpotifyAlbumByID(ctx context.Context, albumID string) ([]core.AlbumCandidate, error) {
	if c == nil {
		return nil, fmt.Errorf("nil chain")
	}
	albumID = strings.TrimSpace(albumID)
	if albumID == "" {
		return nil, core.ErrNoMatch
	}
	if !c.enableSpotify || c.spotifyClientID == "" || c.spotifyClientSecret == "" {
		return nil, core.ErrAllProvidersExhausted
	}
	result, err := c.spBrk.Execute(func() (any, error) {
		tok, err := c.spotifyAccessToken(ctx)
		if err != nil {
			return nil, err
		}
		u := spotifyAlbumURL + url.PathEscape(albumID)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Accept", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusNotFound {
			if c.log != nil {
				c.log.Info("spotify album by id", "spotify_op", "get_album", "outcome", "not_found")
			}
			return nil, core.ErrNoMatch
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if c.log != nil {
				c.log.Warn("spotify album by id", "spotify_op", "get_album", "outcome", "http_error", "status", resp.StatusCode)
			}
			return nil, fmt.Errorf("spotify album http %d", resp.StatusCode)
		}
		cands, err := parseSpotifyAlbumDetailJSON(body)
		if err != nil {
			return nil, err
		}
		if len(cands) == 0 {
			return nil, core.ErrNoMatch
		}
		return cands, nil
	})
	if err != nil {
		if c.log != nil && !errors.Is(err, core.ErrNoMatch) {
			c.log.Warn("spotify album by id", "spotify_op", "get_album", "outcome", "error", "err", err)
		}
		return nil, err
	}
	if c.log != nil {
		c.log.Info("spotify album by id", "spotify_op", "get_album", "outcome", "ok")
	}
	return result.([]core.AlbumCandidate), nil
}

// ResolveSpotifyShareURL implements [core.MetadataOrchestrator] for supported HTTPS share links.
func (c *Chain) ResolveSpotifyShareURL(ctx context.Context, shareURL string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("nil chain")
	}
	if !c.enableSpotify {
		return "", core.ErrAllProvidersExhausted
	}
	if c.spotifyClientID == "" || c.spotifyClientSecret == "" {
		return "", core.ErrAllProvidersExhausted
	}
	id, err := resolveSpotifyShareToAlbumID(ctx, shareURL)
	if err != nil {
		if c.log != nil {
			c.log.Info("spotify share resolve", "spotify_op", "share_redirect", "outcome", "fail")
		}
		return "", err
	}
	if c.log != nil {
		c.log.Info("spotify share resolve", "spotify_op", "share_redirect", "outcome", "ok")
	}
	return id, nil
}
