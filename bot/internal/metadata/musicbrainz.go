package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
)

func (c *Chain) runMusicBrainzSearch(ctx context.Context, q string) ([]core.AlbumCandidate, error) {
	if c == nil {
		return nil, fmt.Errorf("nil chain")
	}
	result, err := c.mbBrk.Execute(func() (any, error) {
		c.mbThrottle.Wait()
		u, err := url.Parse("https://musicbrainz.org/ws/2/release")
		if err != nil {
			return nil, err
		}
		qs := u.Query()
		qs.Set("query", q)
		qs.Set("fmt", "json")
		qs.Set("limit", "3")
		u.RawQuery = qs.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		if c.MusicBrainzUserAgent == "" {
			return nil, fmt.Errorf("empty MusicBrainz User-Agent")
		}
		req.Header.Set("User-Agent", c.MusicBrainzUserAgent)
		req.Header.Set("Accept", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("mb http %d", resp.StatusCode)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("mb http %d", resp.StatusCode)
		}
		return parseMBReleasesJSON(body)
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.([]core.AlbumCandidate), nil
}

type mbJSON struct {
	Releases []struct {
		ID             string         `json:"id"`
		Title          string         `json:"title"`
		Date           string         `json:"date"`
		AC             []mbCredit     `json:"artist-credit"`
		Disambiguation string         `json:"disambiguation"`
	} `json:"releases"`
}

type mbCredit struct {
	Name   string `json:"name"`
	Artist *struct {
		Name string `json:"name"`
	} `json:"artist"`
}

func parseMBReleasesJSON(b []byte) ([]core.AlbumCandidate, error) {
	var w mbJSON
	if err := json.Unmarshal(b, &w); err != nil {
		return nil, err
	}
	if len(w.Releases) == 0 {
		return nil, nil
	}
	out := make([]core.AlbumCandidate, 0, len(w.Releases))
	for i, r := range w.Releases {
		title := r.Title
		if title == "" {
			continue
		}
		artist := artistFromMBCredit(r.AC)
		rel := 1.0 - float64(i)*0.05
		if rel < 0.5 {
			rel = 0.5
		}
		year := yearFromYMD(r.Date)
		c := core.AlbumCandidate{
			Title:         title,
			PrimaryArtist: artist,
			Year:          year,
			Relevance:     rel,
			Provider:      "musicbrainz",
			ProviderRef:   r.ID,
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func artistFromMBCredit(ac []mbCredit) string {
	if len(ac) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ac))
	for _, a := range ac {
		n := a.Name
		if n == "" && a.Artist != nil {
			n = a.Artist.Name
		}
		if n != "" {
			parts = append(parts, n)
		}
	}
	return strings.Join(parts, ", ")
}

func yearFromYMD(s string) *int {
	if len(s) < 4 {
		return nil
	}
	var y int
	_, err := fmt.Sscanf(s[:4], "%d", &y)
	if err != nil || y < 1 {
		return nil
	}
	return &y
}
