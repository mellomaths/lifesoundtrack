package ports

import "context"

// TelegramRecipient is a user with at least one album and a Telegram identity.
type TelegramRecipient struct {
	InternalUserID int64
	TelegramChatID int64 // Private chat id equals Telegram user id.
}

// PickedRecommendation is one fair-random album row for daily send.
type PickedRecommendation struct {
	AlbumInterestID int64
	UserID          int64
	AlbumTitle      string
	Artist          string
}

// RecordRecommendationInput persists the send in DB (album row + audit).
type RecordRecommendationInput struct {
	AlbumInterestID int64
	UserID          int64
	AlbumTitle      string
	Artist          string
}

// RecommendationStore selects albums for daily recommendations and records sends.
type RecommendationStore interface {
	ListTelegramRecipientsWithAlbums(ctx context.Context) ([]TelegramRecipient, error)
	PickNextRecommendation(ctx context.Context, userID int64) (*PickedRecommendation, error)
	RecordRecommendation(ctx context.Context, in RecordRecommendationInput) error
}
