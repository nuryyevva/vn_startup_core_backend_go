package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"vn_startup_core_backend_go/internal/db/sqlc"
)

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(pool)}
}

func (r *PostgresRepository) GetProfile(ctx context.Context, userID uuid.UUID) (Profile, error) {
	row, err := r.queries.GetUserWithProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Profile{}, ErrProfileNotFound
		}
		return Profile{}, fmt.Errorf("get user with profile: %w", err)
	}
	return toDomainProfile(row), nil
}

func (r *PostgresRepository) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (Profile, error) {
	params := sqlc.UpdateUserProfileParams{
		UserID:                  userID,
		DisplayName:             textFromPtr(input.DisplayName),
		Gender:                  textFromPtr(input.Gender),
		Theme:                   textFromPtr(input.Theme),
		Language:                textFromPtr(input.Language),
		FavoriteGenres:          input.FavoriteGenres,
		NotifyNewChapters:       boolFromPtr(input.NotifyNewChapters),
		NotifyPromotionalOffers: boolFromPtr(input.NotifyPromotionalOffers),
	}

	updated, err := r.queries.UpdateUserProfile(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Profile{}, ErrProfileNotFound
		}
		return Profile{}, fmt.Errorf("update user profile: %w", err)
	}

	// UpdateUserProfile doesn't return email/role, so fetch the full joined
	// view to keep the response shape consistent with GetProfile.
	_ = updated
	return r.GetProfile(ctx, userID)
}

func textFromPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func boolFromPtr(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}

func ptrFromText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func toDomainProfile(row sqlc.GetUserWithProfileRow) Profile {
	return Profile{
		UserID:                  row.ID,
		Email:                   row.Email,
		Role:                    row.Role,
		CreatedAt:               row.CreatedAt,
		DisplayName:             ptrFromText(row.DisplayName),
		Gender:                  ptrFromText(row.Gender),
		FavoriteGenres:          row.FavoriteGenres,
		Theme:                   row.Theme,
		Language:                row.Language,
		NotifyNewChapters:       row.NotifyNewChapters,
		NotifyPromotionalOffers: row.NotifyPromotionalOffers,
		UpdatedAt:               row.UpdatedAt,
	}
}
