package metadata

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func redirectResponse(req *http.Request, location string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusFound,
		Header:     http.Header{"Location": []string{location}},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}
}

func okBodyResponse(req *http.Request) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       io.NopCloser(strings.NewReader("ok")),
		Request:    req,
	}
}

func TestResolveSpotifyShareWithHTTPClient_HappyPath(t *testing.T) {
	t.Parallel()
	validAlbum := "https://open.spotify.com/album/1nxWhrFfLczBxMIO80pqNr"
	client := &http.Client{
		CheckRedirect: spotifyShareRedirectPolicy(),
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "spoti.fi" {
				return redirectResponse(req, validAlbum), nil
			}
			if strings.HasPrefix(req.URL.Host, "open.spotify.com") {
				return okBodyResponse(req), nil
			}
			return nil, fmt.Errorf("unexpected host %q", req.URL.Host)
		}),
	}
	id, err := resolveSpotifyShareWithHTTPClient(context.Background(), "https://spoti.fi/xyz", client)
	if err != nil {
		t.Fatal(err)
	}
	if want := "1nxWhrFfLczBxMIO80pqNr"; id != want {
		t.Fatalf("id %q want %q", id, want)
	}
}

func TestResolveSpotifyShareWithHTTPClient_DisallowedRedirectHost(t *testing.T) {
	t.Parallel()
	client := &http.Client{
		CheckRedirect: spotifyShareRedirectPolicy(),
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "spoti.fi" {
				return redirectResponse(req, "https://evil.example/not-spotify"), nil
			}
			return okBodyResponse(req), nil
		}),
	}
	_, err := resolveSpotifyShareWithHTTPClient(context.Background(), "https://spoti.fi/xyz", client)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "redirect") && !strings.Contains(err.Error(), "allowed") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestResolveSpotifyShareWithHTTPClient_NonHTTPSRedirect(t *testing.T) {
	t.Parallel()
	client := &http.Client{
		CheckRedirect: spotifyShareRedirectPolicy(),
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return redirectResponse(req, "http://open.spotify.com/album/1nxWhrFfLczBxMIO80pqNr"), nil
		}),
	}
	_, err := resolveSpotifyShareWithHTTPClient(context.Background(), "https://spoti.fi/xyz", client)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "https") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestResolveSpotifyShareWithHTTPClient_TooManyRedirects(t *testing.T) {
	t.Parallel()
	step := 0
	client := &http.Client{
		CheckRedirect: spotifyShareRedirectPolicy(),
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			step++
			next := fmt.Sprintf("https://open.spotify.com/ephemeral/%d", step)
			return redirectResponse(req, next), nil
		}),
	}
	_, err := resolveSpotifyShareWithHTTPClient(context.Background(), "https://spoti.fi/xyz", client)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestResolveSpotifyShareWithHTTPClient_FinalNotAlbum(t *testing.T) {
	t.Parallel()
	final := "https://open.spotify.com/track/1nxWhrFfLczBxMIO80pqNr"
	client := &http.Client{
		CheckRedirect: spotifyShareRedirectPolicy(),
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "spoti.fi" {
				return redirectResponse(req, final), nil
			}
			return okBodyResponse(req), nil
		}),
	}
	_, err := resolveSpotifyShareWithHTTPClient(context.Background(), "https://spoti.fi/xyz", client)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveSpotifyShareWithHTTPClient_UnsupportedInitialHost(t *testing.T) {
	t.Parallel()
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return okBodyResponse(req), nil
	})}
	_, err := resolveSpotifyShareWithHTTPClient(context.Background(), "https://example.com/x", client)
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("got %v", err)
	}
}
