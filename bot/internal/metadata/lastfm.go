package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
)

func (c *Chain) runLastfmSearch(ctx context.Context, q string) ([]core.AlbumCandidate, error) {
	if c == nil || c.lastfmKey == "" {
		return nil, nil
	}
	result, err := c.lfBrk.Execute(func() (any, error) {
		u, err := url.Parse("https://ws.audioscrobbler.com/2.0/")
		if err != nil {
			return nil, err
		}
		qs := u.Query()
		qs.Set("method", "album.search")
		qs.Set("album", q)
		qs.Set("api_key", c.lastfmKey)
		qs.Set("format", "json")
		qs.Set("limit", "3")
		u.RawQuery = qs.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("lastfm http %d", resp.StatusCode)
		}
		return parseLastfmSearchJSON(body)
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.([]core.AlbumCandidate), nil
}

type lfJSON struct {
	Results struct {
		AlbumMatches struct {
			Album []struct {
				Name   string `json:"name"`
				Artist string `json:"artist"`
				Mbid   string `json:"mbid"`
			} `json:"album"`
		} `json:"albummatches"`
	} `json:"results"`
}

func parseLastfmSearchJSON(b []byte) ([]core.AlbumCandidate, error) {
	var w lfJSON
	if err := json.Unmarshal(b, &w); err != nil {
		return nil, err
	}
	albs := w.Results.AlbumMatches.Album
	if len(albs) == 0 {
		return nil, nil
	}
	out := make([]core.AlbumCandidate, 0, len(albs))
	for i, a := range albs {
		if a.Name == "" {
			continue
		}
		rel := 0.9 - float64(i)*0.05
		if rel < 0.4 {
			rel = 0.4
		}
		ref := a.Mbid
		if ref == "" {
			ref = a.Name + "|" + a.Artist
		}
		out = append(out, core.AlbumCandidate{
			Title:         a.Name,
			PrimaryArtist: a.Artist,
			Relevance:     rel,
			Provider:      "lastfm",
			ProviderRef:   ref,
		})
	}
	return out, nil
}
