package core

import (
	"strings"
)

// ParseAlbumLine returns the free-form /album text. ok is true when the line starts
// with /album (optionally @bot). query may be empty (user sent /album with no text).
func ParseAlbumLine(text string) (query string, ok bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", false
	}
	first := fields[0]
	if !strings.HasPrefix(first, "/") {
		return "", false
	}
	name := strings.TrimPrefix(first, "/")
	if i := strings.Index(name, "@"); i >= 0 {
		name = name[:i]
	}
	if !strings.EqualFold(name, "album") {
		return "", false
	}
	if len(fields) < 2 {
		return "", true
	}
	rest := strings.TrimSpace(text[len(first):])
	return strings.TrimSpace(rest), true
}

// OneBasedPickFromText is true for a private reply that is only "1", "2", or "3".
func OneBasedPickFromText(text string) (n int, ok bool) {
	s := strings.TrimSpace(text)
	if len(s) != 1 {
		return 0, false
	}
	switch s[0] {
	case '1':
		return 1, true
	case '2':
		return 2, true
	case '3':
		return 3, true
	default:
		return 0, false
	}
}
