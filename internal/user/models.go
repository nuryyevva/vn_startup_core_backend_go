package user

import (
	"time"

	"github.com/google/uuid"
)

// Profile is the combined identity + personalization view returned by
// GET /users/me.
type Profile struct {
	UserID                  uuid.UUID
	Email                   string
	Role                    string
	CreatedAt               time.Time
	DisplayName             *string
	Gender                  *string
	FavoriteGenres          []string
	Theme                   string
	Language                string
	NotifyNewChapters       bool
	NotifyPromotionalOffers bool
	UpdatedAt               time.Time
}

// UpdateProfileInput carries only the fields the caller supplied; nil means
// "leave unchanged" everywhere except FavoriteGenres, which is replaced
// wholesale when non-nil (there is no meaningful "append" semantics for a
// genre list).
type UpdateProfileInput struct {
	DisplayName             *string
	Gender                  *string
	FavoriteGenres          []string
	Theme                   *string
	Language                *string
	NotifyNewChapters       *bool
	NotifyPromotionalOffers *bool
}

type UpdateProfileRequest struct {
	DisplayName             *string  `json:"display_name"`
	Gender                  *string  `json:"gender"`
	FavoriteGenres          []string `json:"favorite_genres"`
	Theme                   *string  `json:"theme"`
	Language                *string  `json:"language"`
	NotifyNewChapters       *bool    `json:"notify_new_chapters"`
	NotifyPromotionalOffers *bool    `json:"notify_promotional_offers"`
}

type ProfileResponse struct {
	UserID                  uuid.UUID `json:"user_id"`
	Email                   string    `json:"email"`
	Role                    string    `json:"role"`
	CreatedAt               time.Time `json:"created_at"`
	DisplayName             *string   `json:"display_name"`
	Gender                  *string   `json:"gender"`
	FavoriteGenres          []string  `json:"favorite_genres"`
	Theme                   string    `json:"theme"`
	Language                string    `json:"language"`
	NotifyNewChapters       bool      `json:"notify_new_chapters"`
	NotifyPromotionalOffers bool      `json:"notify_promotional_offers"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func toProfileResponse(p Profile) ProfileResponse {
	return ProfileResponse(p)
}
