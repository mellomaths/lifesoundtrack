package core

import (
	"strconv"
	"strings"
)

// ParseRemoveLine returns the remainder after /remove[@bot] when the line is a /remove command.
// ok is true for /remove with or without text; query may be empty.
func ParseRemoveLine(text string) (query string, ok bool) {
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
	if !strings.EqualFold(name, "remove") {
		return "", false
	}
	if len(fields) < 2 {
		return "", true
	}
	rest := strings.TrimSpace(text[len(first):])
	return strings.TrimSpace(rest), true
}

// RemovePickIndexFromText is true for a private reply that is a decimal 1..99 (one or two digits, no sign).
// Used for /remove disambig follow-up; runs before [OneBasedPickFromText] (album 1/2/3 only).
func RemovePickIndexFromText(text string) (n int, ok bool) {
	s := strings.TrimSpace(text)
	if s == "" || len(s) > 2 {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 || v > 99 {
		return 0, false
	}
	return v, true
}
