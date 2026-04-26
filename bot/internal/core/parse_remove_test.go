package core

import (
	"fmt"
	"testing"
)

func TestParseRemoveLine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"/remove", "", true},
		{"/remove@bot", "", true},
		{"/remove@MyBot  foo", "foo", true},
		{"/REMOVE  Kind of Blue", "Kind of Blue", true},
		{"/album x", "", false},
		{"/list", "", false},
		{"remove x", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q", tc.in), func(t *testing.T) {
			t.Parallel()
			got, ok := ParseRemoveLine(tc.in)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRemovePickIndexFromText(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in string
		n  int
		ok bool
	}{
		{"1", 1, true},
		{" 12 ", 12, true},
		{"99", 99, true},
		{"100", 0, false},
		{"0", 0, false},
		{"01", 1, true},
		{"abc", 0, false},
		{"1a", 0, false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q", tc.in), func(t *testing.T) {
			t.Parallel()
			n, ok := RemovePickIndexFromText(tc.in)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if n != tc.n {
				t.Fatalf("n = %d, want %d", n, tc.n)
			}
		})
	}
}

func TestNormalizeArtistQueryRemoveMatch(t *testing.T) {
	t.Parallel()
	u := "  KIND  of  Blue "
	s := "kind of blue"
	if NormalizeArtistQuery(u) != NormalizeArtistQuery(s) {
		t.Fatalf("mismatch: %q vs %q", NormalizeArtistQuery(u), NormalizeArtistQuery(s))
	}
}
