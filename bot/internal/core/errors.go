package core

import "errors"

// Sentinel errors for metadata search; implementations in metadata/ wrap these with [fmt.Errorf]
// and %w so [errors.Is] works from core without importing the metadata package.
var (
	ErrNoMatch               = errors.New("no album match")
	ErrAllProvidersExhausted = errors.New("all metadata providers exhausted or unavailable")
)
