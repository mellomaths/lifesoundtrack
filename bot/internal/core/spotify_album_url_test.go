package core

import (
	"net/url"
	"testing"
)

func TestSpotifyAlbumIDFromParsedURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		raw    string
		wantID string
		wantOK bool
		name   string
	}{
		{
			name:   "locale path and si query",
			raw:    "https://open.spotify.com/intl-pt/album/1fneiuP0JUPv6Hy78xLc2g?si=abc",
			wantID: "1fneiuP0JUPv6Hy78xLc2g",
			wantOK: true,
		},
		{
			name:   "simple album path",
			raw:    "https://open.spotify.com/album/1nxWhrFfLczBxMIO80pqNr",
			wantID: "1nxWhrFfLczBxMIO80pqNr",
			wantOK: true,
		},
		{
			name:   "subdomain open",
			raw:    "https://foo.open.spotify.com/album/1nxWhrFfLczBxMIO80pqNr",
			wantID: "1nxWhrFfLczBxMIO80pqNr",
			wantOK: true,
		},
		{
			name:   "track page not album",
			raw:    "https://open.spotify.com/track/1nxWhrFfLczBxMIO80pqNr",
			wantID: "",
			wantOK: false,
		},
		{
			name:   "wrong host",
			raw:    "https://example.com/album/1nxWhrFfLczBxMIO80pqNr",
			wantID: "",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := mustParseURL(t, tt.raw)
			id, ok := SpotifyAlbumIDFromParsedURL(u)
			if ok != tt.wantOK || id != tt.wantID {
				t.Fatalf("SpotifyAlbumIDFromParsedURL(%q) = %q, %v want %q, %v", tt.raw, id, ok, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestPlanSpotifyAlbumQuery(t *testing.T) {
	t.Parallel()
	album1 := "https://open.spotify.com/album/1nxWhrFfLczBxMIO80pqNr"
	album2 := "https://open.spotify.com/album/2nxWhrFfLczBxMIO80pqNr"
	short1 := "https://spoti.fi/abc"
	tests := []struct {
		q    string
		name string
		want SpotifyAlbumQueryPlan
	}{
		{
			name: "embedded single album prose",
			q:    "hey listen " + album1 + " thanks",
			want: SpotifyAlbumQueryPlan{Mode: SpotifyModeDirect, AlbumID: "1nxWhrFfLczBxMIO80pqNr"},
		},
		{
			name: "non spotify url before album still direct",
			q:    "see https://example.com/x and " + album1,
			want: SpotifyAlbumQueryPlan{Mode: SpotifyModeDirect, AlbumID: "1nxWhrFfLczBxMIO80pqNr"},
		},
		{
			name: "single short link",
			q:    short1,
			want: SpotifyAlbumQueryPlan{Mode: SpotifyModeResolveShort, ShareURL: short1},
		},
		{
			name: "www spotify short host",
			q:    "https://www.spotify.com/share/foo",
			want: SpotifyAlbumQueryPlan{Mode: SpotifyModeResolveShort, ShareURL: "https://www.spotify.com/share/foo"},
		},
		{
			name: "two direct albums ambiguous",
			q:    album1 + " " + album2,
			want: SpotifyAlbumQueryPlan{Mode: SpotifyModeAmbiguousMulti},
		},
		{
			name: "direct plus short ambiguous",
			q:    album1 + " " + short1,
			want: SpotifyAlbumQueryPlan{Mode: SpotifyModeAmbiguousMulti},
		},
		{
			name: "two shorts ambiguous",
			q:    short1 + " https://spoti.fi/def",
			want: SpotifyAlbumQueryPlan{Mode: SpotifyModeAmbiguousMulti},
		},
		{
			name: "open spotify track ineligible",
			q:    "https://open.spotify.com/track/1nxWhrFfLczBxMIO80pqNr",
			want: SpotifyAlbumQueryPlan{Mode: SpotifyModeIneligibleSpotifyHost},
		},
		{
			name: "free text no urls",
			q:    "Abbey Road Beatles",
			want: SpotifyAlbumQueryPlan{Mode: SpotifyModeNone},
		},
		{
			name: "generic https only",
			q:    "https://example.com/foo",
			want: SpotifyAlbumQueryPlan{Mode: SpotifyModeNone},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := PlanSpotifyAlbumQuery(tt.q)
			if got.Mode != tt.want.Mode || got.AlbumID != tt.want.AlbumID || got.ShareURL != tt.want.ShareURL {
				t.Fatalf("PlanSpotifyAlbumQuery(%q) = %#v want %#v", tt.q, got, tt.want)
			}
		})
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
