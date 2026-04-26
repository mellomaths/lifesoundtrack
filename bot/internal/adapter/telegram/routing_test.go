package telegram

import (
	"testing"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
)

// privateRoute documents handleMessage branch order: /album, /list, numeric pick, slash commands.
type privateRoute int

const (
	routeAlbum privateRoute = iota
	routeList
	routePick
	routeSlashOrUnknown
)

func classifyPrivateRoute(text string) privateRoute {
	if _, ok := core.ParseAlbumLine(text); ok {
		return routeAlbum
	}
	if _, _, ok := core.ParseListLine(text); ok {
		return routeList
	}
	if _, ok := core.OneBasedPickFromText(text); ok {
		return routePick
	}
	return routeSlashOrUnknown
}

func TestClassifyPrivateRoute_listBeforePick(t *testing.T) {
	t.Parallel()
	if g := classifyPrivateRoute("/list"); g != routeList {
		t.Fatalf("got %v", g)
	}
	if g := classifyPrivateRoute("1"); g != routePick {
		t.Fatalf("got %v", g)
	}
	if g := classifyPrivateRoute("/list Beatles"); g != routeList {
		t.Fatalf("got %v", g)
	}
	// /list 1 is artist filter "1", not album pick
	if g := classifyPrivateRoute("/list 1"); g != routeList {
		t.Fatalf("got %v", g)
	}
	if g := classifyPrivateRoute("/album x"); g != routeAlbum {
		t.Fatalf("got %v", g)
	}
}

func TestParseListCallbackData(t *testing.T) {
	t.Parallel()
	sid := "550e8400-e29b-41d4-a716-446655440000"
	data := "lpl:" + sid + ":2"
	gotSID, page, ok := parseListCallbackData(data)
	if !ok || gotSID != sid || page != 2 {
		t.Fatalf("got %q %d %v", gotSID, page, ok)
	}
	if _, _, ok := parseListCallbackData("apick:1"); ok {
		t.Fatal("expected false")
	}
}
