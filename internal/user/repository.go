package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrProfileNotFound = errors.New("profile not found")

type Repository interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (Profile, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (Profile, error)
}
