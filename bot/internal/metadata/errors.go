package metadata

import "github.com/mellomaths/lifesoundtrack/bot/internal/core"

// Re-export for callers that import metadata only; values match [core] sentinels.
var (
	ErrNoMatch               = core.ErrNoMatch
	ErrAllProvidersExhausted = core.ErrAllProvidersExhausted
)
