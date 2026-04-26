package telegram

import (
	"testing"

	"github.com/mellomaths/lifesoundtrack/bot/internal/core"
)

// privateRoute documents handleMessage branch order: /album, /list, /remove, 1..99 remove pick, 1/2/3 album pick, slash commands.
type privateRoute int

const (
	routeAlbum privateRoute = iota
	routeList
	routeRemove
	routeRemovePick
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
	if _, ok := core.ParseRemoveLine(text); ok {
		return routeRemove
	}
	if _, ok := core.RemovePickIndexFromText(text); ok {
		return routeRemovePick
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
	// "1".."99" are classified as the remove-pick path first; [handleMessage] falls through to /album pick when not a remove session.
	if g := classifyPrivateRoute("1"); g != routeRemovePick {
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

func TestClassifyPrivateRoute_removeAndTwoDigitBeforeAlbumPick(t *testing.T) {
	t.Parallel()
	if g := classifyPrivateRoute("/remove x"); g != routeRemove {
		t.Fatalf("got %v", g)
	}
	if g := classifyPrivateRoute("12"); g != routeRemovePick {
		t.Fatalf("two-digit should classify as remove-pick path before single-digit album pick; got %v", g)
	}
	if g := classifyPrivateRoute("1"); g != routeRemovePick {
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

func TestParseRemovePickCallbackData(t *testing.T) {
	t.Parallel()
	sid := "550e8400-e29b-41d4-a716-446655440000"
	data := "rmp:" + sid + ":2"
	gotSID, n, ok := parseRemovePickCallbackData(data)
	if !ok || gotSID != sid || n != 2 {
		t.Fatalf("got %q %d %v", gotSID, n, ok)
	}
	if len(data) > 64 {
		t.Fatalf("callback_data over 64 bytes: %d", len(data))
	}
	if _, _, ok := parseRemovePickCallbackData("rmp:bad"); ok {
		t.Fatal("expected false for truncated")
	}
	if _, _, ok := parseRemovePickCallbackData("apick:1"); ok {
		t.Fatal("expected false")
	}
}
