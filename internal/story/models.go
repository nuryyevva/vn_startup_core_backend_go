package story

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Story struct {
	ID          uuid.UUID
	Title       string
	Description *string
	CoverURL    *string
	IsPublished bool
	CreatedAt   time.Time
}

type Scene struct {
	ID                uuid.UUID
	StoryID           uuid.UUID
	OrderIndex        int32
	BackgroundURL     *string
	CharacterID       *uuid.UUID
	DialogueScript    json.RawMessage
	FreeDialogEnabled bool
	DialogLimitType   *string
	DialogLimitValue  *int32
	CreatedAt         time.Time
}

type Choice struct {
	ID           uuid.UUID
	SceneID      uuid.UUID
	Text         string
	IsPaid       bool
	CostDiamonds int32
	NextSceneID  *uuid.UUID
}

type Progress struct {
	UserID         uuid.UUID
	StoryID        uuid.UUID
	CurrentSceneID uuid.UUID
	ChoicesMade    []uuid.UUID
	UpdatedAt      time.Time
}

type ChoiceRequest struct {
	ChoiceID uuid.UUID `json:"choice_id"`
}

// --- response DTOs ---

type StoryResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	CoverURL    *string   `json:"cover_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func toStoryResponse(s Story) StoryResponse {
	return StoryResponse{ID: s.ID, Title: s.Title, Description: s.Description, CoverURL: s.CoverURL, CreatedAt: s.CreatedAt}
}

func toStoryResponses(stories []Story) []StoryResponse {
	out := make([]StoryResponse, 0, len(stories))
	for _, s := range stories {
		out = append(out, toStoryResponse(s))
	}
	return out
}

type ChoiceResponse struct {
	ID           uuid.UUID  `json:"id"`
	Text         string     `json:"text"`
	IsPaid       bool       `json:"is_paid"`
	CostDiamonds int32      `json:"cost_diamonds"`
	NextSceneID  *uuid.UUID `json:"next_scene_id,omitempty"`
}

type SceneResponse struct {
	ID                uuid.UUID        `json:"id"`
	StoryID           uuid.UUID        `json:"story_id"`
	OrderIndex        int32            `json:"order_index"`
	BackgroundURL     *string          `json:"background_url,omitempty"`
	CharacterID       *uuid.UUID       `json:"character_id,omitempty"`
	DialogueScript    json.RawMessage  `json:"dialogue_script"`
	FreeDialogEnabled bool             `json:"free_dialog_enabled"`
	DialogLimitType   *string          `json:"dialog_limit_type,omitempty"`
	DialogLimitValue  *int32           `json:"dialog_limit_value,omitempty"`
	Choices           []ChoiceResponse `json:"choices"`
}

func toChoiceResponse(c Choice) ChoiceResponse {
	return ChoiceResponse{ID: c.ID, Text: c.Text, IsPaid: c.IsPaid, CostDiamonds: c.CostDiamonds, NextSceneID: c.NextSceneID}
}

func toSceneResponse(s Scene, choices []Choice) SceneResponse {
	choiceResponses := make([]ChoiceResponse, 0, len(choices))
	for _, c := range choices {
		choiceResponses = append(choiceResponses, toChoiceResponse(c))
	}
	return SceneResponse{
		ID:                s.ID,
		StoryID:           s.StoryID,
		OrderIndex:        s.OrderIndex,
		BackgroundURL:     s.BackgroundURL,
		CharacterID:       s.CharacterID,
		DialogueScript:    s.DialogueScript,
		FreeDialogEnabled: s.FreeDialogEnabled,
		DialogLimitType:   s.DialogLimitType,
		DialogLimitValue:  s.DialogLimitValue,
		Choices:           choiceResponses,
	}
}

type ProgressResponse struct {
	StoryID        uuid.UUID   `json:"story_id"`
	CurrentSceneID uuid.UUID   `json:"current_scene_id"`
	ChoicesMade    []uuid.UUID `json:"choices_made"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

func toProgressResponse(p Progress) ProgressResponse {
	return ProgressResponse{StoryID: p.StoryID, CurrentSceneID: p.CurrentSceneID, ChoicesMade: p.ChoicesMade, UpdatedAt: p.UpdatedAt}
}
