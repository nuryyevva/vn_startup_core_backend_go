package dialog

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *PostgresRepository) CreateSession(ctx context.Context, userID, characterID, sceneID uuid.UUID, limitType string, limitValue *int32) (Session, error) {
	row, err := r.queries.CreateDialogSession(ctx, sqlc.CreateDialogSessionParams{
		UserID:      userID,
		CharacterID: characterID,
		SceneID:     sceneID,
		LimitType:   limitType,
		LimitValue:  int4FromPtr(limitValue),
	})
	if err != nil {
		return Session{}, fmt.Errorf("create dialog session: %w", err)
	}
	return toDomainSession(row), nil
}

func (r *PostgresRepository) GetSession(ctx context.Context, id uuid.UUID) (Session, error) {
	row, err := r.queries.GetDialogSession(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, fmt.Errorf("get dialog session: %w", err)
	}
	return toDomainSession(row), nil
}

func (r *PostgresRepository) IncrementMessageCount(ctx context.Context, id uuid.UUID) (Session, error) {
	row, err := r.queries.IncrementDialogSessionMessageCount(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, fmt.Errorf("increment dialog session message count: %w", err)
	}
	return toDomainSession(row), nil
}

func (r *PostgresRepository) EndSession(ctx context.Context, id uuid.UUID, reason string) (Session, error) {
	row, err := r.queries.EndDialogSession(ctx, sqlc.EndDialogSessionParams{
		ID:        id,
		EndReason: pgtype.Text{String: reason, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// The UPDATE only matches status='active' rows, so ErrNoRows
			// means either the session doesn't exist or it's already
			// ended/interrupted; distinguish the two for the caller.
			if _, getErr := r.queries.GetDialogSession(ctx, id); getErr != nil {
				if errors.Is(getErr, pgx.ErrNoRows) {
					return Session{}, ErrSessionNotFound
				}
				return Session{}, fmt.Errorf("get dialog session: %w", getErr)
			}
			return Session{}, ErrSessionNotActive
		}
		return Session{}, fmt.Errorf("end dialog session: %w", err)
	}
	return toDomainSession(row), nil
}

func (r *PostgresRepository) ListExpiredTimeLimitedSessions(ctx context.Context) ([]Session, error) {
	rows, err := r.queries.ListExpiredTimeLimitedSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list expired time-limited sessions: %w", err)
	}
	out := make([]Session, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainSession(row))
	}
	return out, nil
}

func (r *PostgresRepository) CreateMessage(ctx context.Context, sessionID uuid.UUID, sender, text string, cost int32) (Message, error) {
	row, err := r.queries.CreateDialogMessage(ctx, sqlc.CreateDialogMessageParams{
		SessionID:    sessionID,
		Sender:       sender,
		Text:         text,
		CostDiamonds: cost,
	})
	if err != nil {
		return Message{}, fmt.Errorf("create dialog message: %w", err)
	}
	return toDomainMessage(row), nil
}

func (r *PostgresRepository) ListMessages(ctx context.Context, sessionID uuid.UUID) ([]Message, error) {
	rows, err := r.queries.ListDialogMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list dialog messages: %w", err)
	}
	out := make([]Message, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainMessage(row))
	}
	return out, nil
}

func (r *PostgresRepository) CountUserMessages(ctx context.Context, userID uuid.UUID, sender string) (int64, error) {
	count, err := r.queries.CountUserDialogMessagesBySender(ctx, sqlc.CountUserDialogMessagesBySenderParams{
		UserID: userID,
		Sender: sender,
	})
	if err != nil {
		return 0, fmt.Errorf("count user dialog messages by sender: %w", err)
	}
	return count, nil
}

func int4FromPtr(v *int32) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *v, Valid: true}
}

func int4ToPtr(v pgtype.Int4) *int32 {
	if !v.Valid {
		return nil
	}
	return &v.Int32
}

func toDomainSession(s sqlc.DialogSession) Session {
	var endedAt *time.Time
	if s.EndedAt.Valid {
		endedAt = &s.EndedAt.Time
	}
	var endReason *string
	if s.EndReason.Valid {
		endReason = &s.EndReason.String
	}
	return Session{
		ID:           s.ID,
		UserID:       s.UserID,
		CharacterID:  s.CharacterID,
		SceneID:      s.SceneID,
		Status:       s.Status,
		LimitType:    s.LimitType,
		LimitValue:   int4ToPtr(s.LimitValue),
		MessageCount: s.MessageCount,
		StartedAt:    s.StartedAt,
		EndedAt:      endedAt,
		EndReason:    endReason,
	}
}

func toDomainMessage(m sqlc.DialogMessage) Message {
	return Message{
		ID:           m.ID,
		SessionID:    m.SessionID,
		Sender:       m.Sender,
		Text:         m.Text,
		CostDiamonds: m.CostDiamonds,
		CreatedAt:    m.CreatedAt,
	}
}
