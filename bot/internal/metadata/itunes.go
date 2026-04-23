package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
)

func (c *Chain) runITunesSearch(ctx context.Context, q string) ([]core.AlbumCandidate, error) {
	if c == nil {
		return nil, fmt.Errorf("nil chain")
	}
	result, err := c.itBrk.Execute(func() (any, error) {
		u, err := url.Parse("https://itunes.apple.com/search")
		if err != nil {
			return nil, err
		}
		qs := u.Query()
		qs.Set("term", q)
		qs.Set("entity", "album")
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
			return nil, fmt.Errorf("itunes http %d", resp.StatusCode)
		}
		return parseITunesSearchJSON(body)
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.([]core.AlbumCandidate), nil
}

type itJSON struct {
	Results []struct {
		CollectionName string `json:"collectionName"`
		ArtistName     string `json:"artistName"`
		ReleaseDate    string `json:"releaseDate"`
		ArtworkURL     string `json:"artworkUrl100"`
		CollectionID   int64  `json:"collectionId"`
	} `json:"results"`
}

func parseITunesSearchJSON(b []byte) ([]core.AlbumCandidate, error) {
	var w itJSON
	if err := json.Unmarshal(b, &w); err != nil {
		return nil, err
	}
	if len(w.Results) == 0 {
		return nil, nil
	}
	out := make([]core.AlbumCandidate, 0, len(w.Results))
	for i, r := range w.Results {
		if r.CollectionName == "" {
			continue
		}
		rel := 0.85 - float64(i)*0.05
		if rel < 0.35 {
			rel = 0.35
		}
		art := r.ArtworkURL
		if strings.HasPrefix(art, "http://") {
			art = "https://" + strings.TrimPrefix(art, "http://")
		}
		year := yearFromRFC3339Date(r.ReleaseDate)
		out = append(out, core.AlbumCandidate{
			Title:         r.CollectionName,
			PrimaryArtist: r.ArtistName,
			Year:          year,
			Relevance:     rel,
			Provider:      "itunes",
			ProviderRef:   fmt.Sprintf("%d", r.CollectionID),
			ArtURL:        art,
		})
	}
	return out, nil
}

func yearFromRFC3339Date(s string) *int {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	y := t.Year()
	if y < 1 {
		return nil
	}
	return &y
}
