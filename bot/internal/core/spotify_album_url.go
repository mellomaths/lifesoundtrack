package core

import (
	"net/url"
	"regexp"
	"strings"
)

// Short share hosts treated as FR-008 short links (research §2).
func isSpotifyShortShareHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	return h == "spoti.fi" || h == "www.spotify.com"
}

var httpURLRegexp = regexp.MustCompile(`https?://[^\s]+`)

// extractHTTPLikeURLs finds candidate http(s) URLs in s (trim trailing punctuation).
func extractHTTPLikeURLs(s string) []string {
	raw := httpURLRegexp.FindAllString(s, -1)
	out := make([]string, 0, len(raw))
	trimTrail := func(u string) string {
		return strings.TrimRight(u, ".,;:!?)]}\"'")
	}
	for _, m := range raw {
		m = trimTrail(m)
		if m != "" {
			out = append(out, m)
		}
	}
	return out
}

// SpotifyAlbumIDFromParsedURL extracts the album id when u is an open.spotify.com album page (shared with metadata redirect resolution).
func SpotifyAlbumIDFromParsedURL(u *url.URL) (id string, ok bool) {
	return spotifyAlbumIDFromURL(u)
}

// spotifyAlbumIDFromURL returns the album id when u is an open.spotify.com album page.
func spotifyAlbumIDFromURL(u *url.URL) (id string, ok bool) {
	if u == nil {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	if host != "open.spotify.com" && !strings.HasSuffix(host, ".open.spotify.com") {
		return "", false
	}
	segs := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := 0; i < len(segs)-1; i++ {
		if strings.EqualFold(segs[i], "album") {
			id = strings.TrimSpace(segs[i+1])
			if len(id) >= 10 {
				return id, true
			}
			return "", false
		}
	}
	return "", false
}

// AnalyzeSpotifyAlbumQuery extracts distinct direct album IDs and short-share URLs from the full user argument.
func AnalyzeSpotifyAlbumQuery(q string) (directAlbumIDs []string, shortShareURLs []string) {
	seenID := make(map[string]struct{})
	for _, raw := range extractHTTPLikeURLs(q) {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		host := strings.ToLower(u.Hostname())
		if host == "open.spotify.com" || strings.HasSuffix(host, ".open.spotify.com") {
			if id, ok := spotifyAlbumIDFromURL(u); ok {
				if _, dup := seenID[id]; !dup {
					seenID[id] = struct{}{}
					directAlbumIDs = append(directAlbumIDs, id)
				}
			}
			continue
		}
		if isSpotifyShortShareHost(host) {
			shortShareURLs = append(shortShareURLs, raw)
		}
	}
	return directAlbumIDs, shortShareURLs
}

// SpotifyAlbumQueryPlan describes how SaveService should handle FR-008 input.
type SpotifyAlbumQueryPlan struct {
	// Mode is none (use Search), directAlbumID, resolveShort, ambiguous, or ineligibleSpotifyPage.
	Mode SpotifyAlbumQueryMode
	// AlbumID is set when Mode == SpotifyModeDirect.
	AlbumID string
	// ShareURL is set when Mode == SpotifyModeResolveShort.
	ShareURL string
}

// SpotifyAlbumQueryMode is the classification for link-shaped input.
type SpotifyAlbumQueryMode int

const (
	SpotifyModeNone SpotifyAlbumQueryMode = iota
	SpotifyModeDirect
	SpotifyModeResolveShort
	SpotifyModeAmbiguousMulti
	SpotifyModeIneligibleSpotifyHost
)

// PlanSpotifyAlbumQuery maps extracted URLs to a single plan per research §1 / spec Edge Cases.
func PlanSpotifyAlbumQuery(q string) SpotifyAlbumQueryPlan {
	direct, shorts := AnalyzeSpotifyAlbumQuery(q)
	switch {
	case len(direct) >= 2:
		return SpotifyAlbumQueryPlan{Mode: SpotifyModeAmbiguousMulti}
	case len(direct) == 1 && len(shorts) >= 1:
		return SpotifyAlbumQueryPlan{Mode: SpotifyModeAmbiguousMulti}
	case len(direct) == 1:
		return SpotifyAlbumQueryPlan{Mode: SpotifyModeDirect, AlbumID: direct[0]}
	case len(shorts) >= 2:
		return SpotifyAlbumQueryPlan{Mode: SpotifyModeAmbiguousMulti}
	case len(shorts) == 1:
		return SpotifyAlbumQueryPlan{Mode: SpotifyModeResolveShort, ShareURL: shorts[0]}
	}
	if hasIneligibleOpenSpotifyURL(q) {
		return SpotifyAlbumQueryPlan{Mode: SpotifyModeIneligibleSpotifyHost}
	}
	return SpotifyAlbumQueryPlan{Mode: SpotifyModeNone}
}

func hasIneligibleOpenSpotifyURL(q string) bool {
	for _, raw := range extractHTTPLikeURLs(q) {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		host := strings.ToLower(u.Hostname())
		if host == "open.spotify.com" || strings.HasSuffix(host, ".open.spotify.com") {
			if _, ok := spotifyAlbumIDFromURL(u); !ok {
				return true
			}
		}
	}
	return false
}
