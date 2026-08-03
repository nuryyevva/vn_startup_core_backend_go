package story

import (
	"context"
	"encoding/json"
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

func (r *PostgresRepository) ListPublishedStories(ctx context.Context) ([]Story, error) {
	rows, err := r.queries.ListPublishedStories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list published stories: %w", err)
	}
	out := make([]Story, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainStory(row))
	}
	return out, nil
}

func (r *PostgresRepository) GetStory(ctx context.Context, id uuid.UUID) (Story, error) {
	row, err := r.queries.GetStory(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Story{}, ErrStoryNotFound
		}
		return Story{}, fmt.Errorf("get story: %w", err)
	}
	return toDomainStory(row), nil
}

func (r *PostgresRepository) GetFirstScene(ctx context.Context, storyID uuid.UUID) (Scene, error) {
	row, err := r.queries.GetFirstSceneOfStory(ctx, storyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scene{}, ErrSceneNotFound
		}
		return Scene{}, fmt.Errorf("get first scene of story: %w", err)
	}
	return toDomainScene(row), nil
}

func (r *PostgresRepository) GetScene(ctx context.Context, id uuid.UUID) (Scene, error) {
	row, err := r.queries.GetScene(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scene{}, ErrSceneNotFound
		}
		return Scene{}, fmt.Errorf("get scene: %w", err)
	}
	return toDomainScene(row), nil
}

func (r *PostgresRepository) ListChoices(ctx context.Context, sceneID uuid.UUID) ([]Choice, error) {
	rows, err := r.queries.ListChoicesByScene(ctx, sceneID)
	if err != nil {
		return nil, fmt.Errorf("list choices by scene: %w", err)
	}
	out := make([]Choice, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainChoice(row))
	}
	return out, nil
}

func (r *PostgresRepository) GetChoice(ctx context.Context, id uuid.UUID) (Choice, error) {
	row, err := r.queries.GetChoice(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Choice{}, ErrChoiceNotFound
		}
		return Choice{}, fmt.Errorf("get choice: %w", err)
	}
	return toDomainChoice(row), nil
}

func (r *PostgresRepository) GetProgress(ctx context.Context, userID, storyID uuid.UUID) (Progress, error) {
	row, err := r.queries.GetPlayerProgress(ctx, sqlc.GetPlayerProgressParams{UserID: userID, StoryID: storyID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Progress{}, ErrProgressNotFound
		}
		return Progress{}, fmt.Errorf("get player progress: %w", err)
	}
	return toDomainProgress(row)
}

func (r *PostgresRepository) CreateProgress(ctx context.Context, userID, storyID, sceneID uuid.UUID) (Progress, error) {
	row, err := r.queries.CreatePlayerProgress(ctx, sqlc.CreatePlayerProgressParams{
		UserID:         userID,
		StoryID:        storyID,
		CurrentSceneID: sceneID,
	})
	if err != nil {
		return Progress{}, fmt.Errorf("create player progress: %w", err)
	}
	return toDomainProgress(row)
}

func (r *PostgresRepository) UpdateProgress(ctx context.Context, userID, storyID, sceneID uuid.UUID, choicesMade []uuid.UUID) (Progress, error) {
	choicesJSON, err := json.Marshal(choicesMade)
	if err != nil {
		return Progress{}, fmt.Errorf("marshal choices_made: %w", err)
	}

	row, err := r.queries.UpdatePlayerProgress(ctx, sqlc.UpdatePlayerProgressParams{
		UserID:         userID,
		StoryID:        storyID,
		CurrentSceneID: sceneID,
		ChoicesMade:    choicesJSON,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Progress{}, ErrProgressNotFound
		}
		return Progress{}, fmt.Errorf("update player progress: %w", err)
	}
	return toDomainProgress(row)
}

func toDomainStory(s sqlc.Story) Story {
	return Story{
		ID:          s.ID,
		Title:       s.Title,
		Description: ptrFromText(s.Description),
		CoverURL:    ptrFromText(s.CoverUrl),
		IsPublished: s.IsPublished,
		CreatedAt:   s.CreatedAt,
	}
}

func toDomainScene(s sqlc.Scene) Scene {
	return Scene{
		ID:                s.ID,
		StoryID:           s.StoryID,
		OrderIndex:        s.OrderIndex,
		BackgroundURL:     ptrFromText(s.BackgroundUrl),
		CharacterID:       ptrFromUUID(s.CharacterID),
		DialogueScript:    json.RawMessage(s.DialogueScript),
		FreeDialogEnabled: s.FreeDialogEnabled,
		DialogLimitType:   ptrFromText(s.DialogLimitType),
		DialogLimitValue:  ptrFromInt4(s.DialogLimitValue),
		CreatedAt:         s.CreatedAt,
	}
}

func toDomainChoice(c sqlc.Choice) Choice {
	return Choice{
		ID:           c.ID,
		SceneID:      c.SceneID,
		Text:         c.Text,
		IsPaid:       c.IsPaid,
		CostDiamonds: c.CostDiamonds,
		NextSceneID:  ptrFromUUID(c.NextSceneID),
	}
}

func toDomainProgress(p sqlc.PlayerProgress) (Progress, error) {
	var choicesMade []uuid.UUID
	if len(p.ChoicesMade) > 0 {
		if err := json.Unmarshal(p.ChoicesMade, &choicesMade); err != nil {
			return Progress{}, fmt.Errorf("unmarshal choices_made: %w", err)
		}
	}
	return Progress{
		UserID:         p.UserID,
		StoryID:        p.StoryID,
		CurrentSceneID: p.CurrentSceneID,
		ChoicesMade:    choicesMade,
		UpdatedAt:      p.UpdatedAt,
	}, nil
}

func ptrFromText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func ptrFromInt4(i pgtype.Int4) *int32 {
	if !i.Valid {
		return nil
	}
	return &i.Int32
}

func ptrFromUUID(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}
