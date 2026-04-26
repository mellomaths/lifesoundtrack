package core

import "strings"

// ListPageSize is albums per /list page (contracts/list-command.md).
const ListPageSize = 5

// ListParseKind classifies a private /list line.
type ListParseKind int

const (
	// ListParseNotList means the line is not a /list command.
	ListParseNotList ListParseKind = iota
	// ListParseBareOrWhitespace is /list with no artist filter.
	ListParseBareOrWhitespace
	// ListParseArtistFilter is /list <remainder> with non-empty normalized filter.
	ListParseArtistFilter
	// ListParseNext is /list next (case-insensitive, exact remainder).
	ListParseNext
	// ListParseBack is /list back.
	ListParseBack
)

// NormalizeArtistQuery trims Unicode space, collapses internal whitespace (strings.Fields), lowercases.
func NormalizeArtistQuery(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	fields := strings.Fields(s)
	return strings.ToLower(strings.Join(fields, " "))
}

// ParseListLine returns whether text starts with /list[@bot]; when ok, kind and optional artist remainder (raw, before normalization).
func ParseListLine(text string) (kind ListParseKind, artistRemainder string, ok bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return ListParseNotList, "", false
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ListParseNotList, "", false
	}
	first := fields[0]
	if !strings.HasPrefix(first, "/") {
		return ListParseNotList, "", false
	}
	name := strings.TrimPrefix(first, "/")
	if i := strings.Index(name, "@"); i >= 0 {
		name = name[:i]
	}
	if !strings.EqualFold(name, "list") {
		return ListParseNotList, "", false
	}
	ok = true
	if len(fields) == 1 {
		return ListParseBareOrWhitespace, "", true
	}
	rest := strings.TrimSpace(text[len(first):])
	if rest == "" {
		return ListParseBareOrWhitespace, "", true
	}
	if strings.EqualFold(rest, "next") {
		return ListParseNext, "", true
	}
	if strings.EqualFold(rest, "back") {
		return ListParseBack, "", true
	}
	return ListParseArtistFilter, rest, true
}
