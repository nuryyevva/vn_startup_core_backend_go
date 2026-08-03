package user

import (
	"time"

	"github.com/google/uuid"
)

// Profile is the combined identity + personalization view returned by
// GET /users/me.
type Profile struct {
	UserID      uuid.UUID
	Email       string
	Role        string
	DisplayName *string
	Gender      *string
	Hobbies     []string
	Theme       string
	Language    string
	UpdatedAt   time.Time
}

// UpdateProfileInput carries only the fields the caller supplied; nil means
// "leave unchanged" everywhere except Hobbies, which is replaced wholesale
// when non-nil (there is no meaningful "append" semantics for a hobby list).
type UpdateProfileInput struct {
	DisplayName *string
	Gender      *string
	Hobbies     []string
	Theme       *string
	Language    *string
}

type UpdateProfileRequest struct {
	DisplayName *string  `json:"display_name"`
	Gender      *string  `json:"gender"`
	Hobbies     []string `json:"hobbies"`
	Theme       *string  `json:"theme"`
	Language    *string  `json:"language"`
}

type ProfileResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	DisplayName *string   `json:"display_name"`
	Gender      *string   `json:"gender"`
	Hobbies     []string  `json:"hobbies"`
	Theme       string    `json:"theme"`
	Language    string    `json:"language"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toProfileResponse(p Profile) ProfileResponse {
	return ProfileResponse(p)
}
