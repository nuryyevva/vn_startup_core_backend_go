package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already registered")
)

// Repository persists and retrieves user accounts. Registering a user also
// creates its default profile row so every user always has exactly one
// user_profiles record.
type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error
}
