package core

import (
	"encoding/json"
	"fmt"
	"time"
)

const maxExtraJSONBytes = 8 * 1024

// EncodeCandidateExtra builds a small JSONB payload for [data-model] extra (no PII, bounded size).
func EncodeCandidateExtra(c *AlbumCandidate) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("nil candidate")
	}
	m := map[string]any{
		"provider":     c.Provider,
		"provider_ref": c.ProviderRef,
		"relevance":    c.Relevance,
		"captured_at":  time.Now().UTC().Format(time.RFC3339),
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	if len(b) > maxExtraJSONBytes {
		return nil, fmt.Errorf("extra json exceeds %d bytes", maxExtraJSONBytes)
	}
	return b, nil
}
