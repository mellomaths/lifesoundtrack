package ports

import "context"

// SourceTelegram identifies Telegram as the external identity provider.
const SourceTelegram = "telegram"

// AddInterestInput persists an album interest for a user resolved via external identity.
type AddInterestInput struct {
	Source      string
	ExternalID  string // Provider user id as decimal string (e.g. Telegram From.ID).
	DisplayName *string
	Username    *string // Telegram @handle without leading @.
	AlbumTitle  string
	Artist      string
}

// InterestWriter persists album interests and associated user / identity rows.
type InterestWriter interface {
	AddInterest(ctx context.Context, in AddInterestInput) error
}
