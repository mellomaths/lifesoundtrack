package core

import (
	"fmt"
	"testing"

	"github.com/mellomaths/lifesoundtrack/bot/internal/store"
)

func TestExactTitleMatches(t *testing.T) {
	t.Parallel()
	rows := []store.SavedAlbumListRow{
		{ID: "a1", Title: "Kind of Blue"},
		{ID: "a2", Title: "KIND  of  Blue"},
		{ID: "a3", Title: "Other"},
	}
	nq := NormalizeArtistQuery("  kind  of  blue  ")
	got := exactTitleMatches(rows, nq)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(got), got)
	}
	empty := exactTitleMatches(rows, "")
	if len(empty) != 0 {
		t.Fatalf("empty nq: got %d matches", len(empty))
	}
}

func TestPartialTitleMatches(t *testing.T) {
	t.Parallel()
	rows := []store.SavedAlbumListRow{
		{ID: "1", Title: "Abbey Road (Remastered)"},
		{ID: "2", Title: "The White Album"},
		{ID: "3", Title: "Help!"},
	}
	cases := []struct {
		query string
		wantN int
		ids   []string
	}{
		{"Abbey Road", 1, []string{"1"}},
		{"abbey  road", 1, []string{"1"}},
		{"remastered", 1, []string{"1"}},
		{"White", 1, []string{"2"}},
		{"nomatch", 0, nil},
	}
	for _, tc := range cases {
		tc := tc
		nq := NormalizeArtistQuery(tc.query)
		t.Run(fmt.Sprintf("%q", tc.query), func(t *testing.T) {
			t.Parallel()
			got := partialTitleMatches(rows, nq)
			if len(got) != tc.wantN {
				t.Fatalf("len = %d, want %d: %+v", len(got), tc.wantN, got)
			}
			if tc.wantN == 0 {
				return
			}
			for i, id := range tc.ids {
				if got[i].ID != id {
					t.Fatalf("got id %q at %d, want %q", got[i].ID, i, id)
				}
			}
		})
	}
}

func TestPartialTitleMatches_fiveContainingToken(t *testing.T) {
	t.Parallel()
	rows := make([]store.SavedAlbumListRow, 4)
	for i := range rows {
		rows[i] = store.SavedAlbumListRow{ID: fmt.Sprintf("id-%d", i), Title: "Album xx token yy"}
	}
	// 4 rows all contain "token" in normalized title
	got := partialTitleMatches(rows, NormalizeArtistQuery("token"))
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}
}
