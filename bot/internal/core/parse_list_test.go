package core

import "testing"

func TestNormalizeArtistQuery(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"  foo  ", "foo"},
		{"FOO", "foo"},
		{"The  Beatles", "the beatles"},
		{"  a \t b \n c  ", "a b c"},
	}
	for _, tc := range cases {
		got := NormalizeArtistQuery(tc.in)
		if got != tc.want {
			t.Fatalf("NormalizeArtistQuery(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseListLine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		text       string
		wantKind   ListParseKind
		wantArtist string
		wantOK     bool
	}{
		{"/album x", ListParseNotList, "", false},
		{"/list", ListParseBareOrWhitespace, "", true},
		{"/list@mybot", ListParseBareOrWhitespace, "", true},
		{"/list   ", ListParseBareOrWhitespace, "", true},
		{"/list \t ", ListParseBareOrWhitespace, "", true},
		{"/list The Beatles", ListParseArtistFilter, "The Beatles", true},
		{"/list@b The Beatles", ListParseArtistFilter, "The Beatles", true},
		{"/list next", ListParseNext, "", true},
		{"/list NEXT", ListParseNext, "", true},
		{"/list back", ListParseBack, "", true},
		{"list", ListParseNotList, "", false},
	}
	for _, tc := range cases {
		k, rest, ok := ParseListLine(tc.text)
		if ok != tc.wantOK || k != tc.wantKind || rest != tc.wantArtist {
			t.Fatalf("ParseListLine(%q) = (%v, %q, %v); want (%v, %q, %v)",
				tc.text, k, rest, ok, tc.wantKind, tc.wantArtist, tc.wantOK)
		}
	}
}
