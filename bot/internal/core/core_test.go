package core

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseTextMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want Command
	}{
		{"/start", CommandStart},
		{"/Start", CommandStart},
		{"/start@mybot", CommandStart},
		{"/start payload", CommandStart},
		{"/help", CommandHelp},
		{"/help@mybot", CommandHelp},
		{"/ping", CommandPing},
		{"/pong", CommandUnknown},
		{"", CommandUnknown},
		{"  ", CommandUnknown},
		{"hi", CommandUnknown},
		{"/unknown", CommandUnknown},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.in), func(t *testing.T) {
			t.Parallel()
			if got := ParseTextMessage(tt.in); got != tt.want {
				t.Errorf("ParseTextMessage(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestReply(t *testing.T) {
	t.Parallel()
	if s := Reply(CommandStart); !strings.Contains(s, productName) {
		t.Errorf("start reply must include %q, got: %q", productName, s)
	}
	if s := Reply(CommandHelp); !strings.Contains(s, productName) {
		t.Errorf("help reply must include %q, got: %q", productName, s)
	}
	help := Reply(CommandHelp)
	if !strings.Contains(help, "/start") || !strings.Contains(help, "/help") || !strings.Contains(help, "/ping") || !strings.Contains(help, "/album") {
		t.Errorf("help must list supported commands, got: %q", help)
	}
	if !strings.Contains(strings.ToLower(help), "spotify") {
		t.Errorf("help should mention Spotify album/share links, got: %q", help)
	}
	start := Reply(CommandStart)
	if !strings.Contains(strings.ToLower(start), "spotify") {
		t.Errorf("start should mention Spotify link paste, got: %q", start)
	}
	if s := Reply(CommandPing); strings.TrimSpace(s) == "" {
		t.Error("ping reply must be non-empty")
	}
	if s := Reply(CommandPing); !strings.Contains(s, "pong") {
		t.Errorf("ping should include pong-style liveness, got: %q", s)
	}
	if s := Reply(CommandUnknown); !strings.Contains(strings.ToLower(s), "/help") {
		t.Errorf("unknown should hint at help, got: %q", s)
	}
}

func TestCommandString(t *testing.T) {
	t.Parallel()
	if CommandStart.String() != "start" {
		t.Errorf("got %q", CommandStart.String())
	}
}

func TestParseAlbumLine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"/album", "", true},
		{"/album@bot", "", true},
		{"/album   ", "", true},
		{"/album Red", "Red", true},
		{"/ALBUM  Red  Hot", "Red  Hot", true},
		{"/start", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		if got, ok := ParseAlbumLine(tc.in); ok != tc.ok || (tc.ok && got != tc.want) {
			t.Errorf("ParseAlbumLine(%q) = %q, %v want %q, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestOneBasedPickFromText(t *testing.T) {
	t.Parallel()
	if n, ok := OneBasedPickFromText("2"); !ok || n != 2 {
		t.Errorf("2: got %d %v", n, ok)
	}
	if _, ok := OneBasedPickFromText("12"); ok {
		t.Error("12 should not be a pick")
	}
}
