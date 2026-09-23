package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vn_startup_core_backend_go/internal/db/sqlc"
)

const pgUniqueViolationCode = "23505"

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// CreateUser inserts the user and its default profile row in a single
// transaction so a user never exists without a profile.
func (r *PostgresRepository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.queries.WithTx(tx)

	dbUser, err := q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode {
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}

	if _, err := q.CreateUserProfile(ctx, dbUser.ID); err != nil {
		return User{}, fmt.Errorf("insert default user profile: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit tx: %w", err)
	}

	return toDomainUser(dbUser), nil
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	dbUser, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("get user by email: %w", err)
	}
	return toDomainUser(dbUser), nil
}

func (r *PostgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	dbUser, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("get user by id: %w", err)
	}
	return toDomainUser(dbUser), nil
}

func (r *PostgresRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	if err := r.queries.UpdateUserPasswordHash(ctx, sqlc.UpdateUserPasswordHashParams{ID: id, PasswordHash: passwordHash}); err != nil {
		return fmt.Errorf("update user password hash: %w", err)
	}
	return nil
}

func toDomainUser(u sqlc.User) User {
	return User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		CreatedAt:    u.CreatedAt,
	}
}
