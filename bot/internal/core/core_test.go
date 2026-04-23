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
	if !strings.Contains(Reply(CommandHelp), "/start") || !strings.Contains(Reply(CommandHelp), "/help") || !strings.Contains(Reply(CommandHelp), "/ping") {
		t.Errorf("help must list three commands, got: %q", Reply(CommandHelp))
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
