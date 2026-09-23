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
	Genre       string
	Status      string
	IsPublished bool
	CreatedAt   time.Time
}

type Scene struct {
	ID                 uuid.UUID
	StoryID            uuid.UUID
	OrderIndex         int32
	BackgroundURL      *string
	CharacterID        *uuid.UUID
	DialogueScript     json.RawMessage
	FreeDialogEnabled  bool
	DialogLimitType    *string
	DialogLimitValue   *int32
	CreatedAt          time.Time
	UnlockCostDiamonds *int32
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
	Genre       string    `json:"genre"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

func toStoryResponse(s Story) StoryResponse {
	return StoryResponse{
		ID:          s.ID,
		Title:       s.Title,
		Description: s.Description,
		CoverURL:    s.CoverURL,
		Genre:       s.Genre,
		Status:      s.Status,
		CreatedAt:   s.CreatedAt,
	}
}

func toStoryResponses(stories []Story) []StoryResponse {
	out := make([]StoryResponse, 0, len(stories))
	for _, s := range stories {
		out = append(out, toStoryResponse(s))
	}
	return out
}

// StoryPage is one page of ListPublishedStories: the resolved (defaulted,
// clamped) page/pageSize the service actually used, plus the total count of
// stories matching the filter so the client knows whether more pages exist.
type StoryPage struct {
	Stories  []Story
	Total    int64
	Page     int32
	PageSize int32
}

type StoryListResponse struct {
	Items    []StoryResponse `json:"items"`
	Total    int64           `json:"total"`
	Page     int32           `json:"page"`
	PageSize int32           `json:"page_size"`
}

func toStoryListResponse(p StoryPage) StoryListResponse {
	return StoryListResponse{
		Items:    toStoryResponses(p.Stories),
		Total:    p.Total,
		Page:     p.Page,
		PageSize: p.PageSize,
	}
}

type ChoiceResponse struct {
	ID           uuid.UUID  `json:"id"`
	Text         string     `json:"text"`
	IsPaid       bool       `json:"is_paid"`
	CostDiamonds int32      `json:"cost_diamonds"`
	NextSceneID  *uuid.UUID `json:"next_scene_id,omitempty"`
}

type SceneResponse struct {
	ID                 uuid.UUID        `json:"id"`
	StoryID            uuid.UUID        `json:"story_id"`
	OrderIndex         int32            `json:"order_index"`
	BackgroundURL      *string          `json:"background_url,omitempty"`
	CharacterID        *uuid.UUID       `json:"character_id,omitempty"`
	DialogueScript     json.RawMessage  `json:"dialogue_script"`
	FreeDialogEnabled  bool             `json:"free_dialog_enabled"`
	DialogLimitType    *string          `json:"dialog_limit_type,omitempty"`
	DialogLimitValue   *int32           `json:"dialog_limit_value,omitempty"`
	UnlockCostDiamonds *int32           `json:"unlock_cost_diamonds,omitempty"`
	Choices            []ChoiceResponse `json:"choices"`
}

// SceneSummaryResponse is a lightweight per-scene entry returned by
// GET /stories/:id/scenes, used to render a story's chapter list (with
// lock state) without fetching each scene's full dialogue script.
type SceneSummaryResponse struct {
	ID                 uuid.UUID `json:"id"`
	OrderIndex         int32     `json:"order_index"`
	UnlockCostDiamonds *int32    `json:"unlock_cost_diamonds,omitempty"`
	IsUnlocked         bool      `json:"is_unlocked"`
}

func toSceneSummaryResponse(s Scene, isUnlocked bool) SceneSummaryResponse {
	return SceneSummaryResponse{
		ID:                 s.ID,
		OrderIndex:         s.OrderIndex,
		UnlockCostDiamonds: s.UnlockCostDiamonds,
		IsUnlocked:         isUnlocked,
	}
}

func toSceneSummaryResponses(scenes []Scene, unlockedSceneIDs map[uuid.UUID]bool) []SceneSummaryResponse {
	out := make([]SceneSummaryResponse, 0, len(scenes))
	for _, s := range scenes {
		isUnlocked := s.UnlockCostDiamonds == nil || unlockedSceneIDs[s.ID]
		out = append(out, toSceneSummaryResponse(s, isUnlocked))
	}
	return out
}

// SceneUnlockResponse confirms a successful POST /scenes/:id/unlock call.
type SceneUnlockResponse struct {
	SceneID  uuid.UUID `json:"scene_id"`
	Unlocked bool      `json:"unlocked"`
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
		ID:                 s.ID,
		StoryID:            s.StoryID,
		OrderIndex:         s.OrderIndex,
		BackgroundURL:      s.BackgroundURL,
		CharacterID:        s.CharacterID,
		DialogueScript:     s.DialogueScript,
		FreeDialogEnabled:  s.FreeDialogEnabled,
		DialogLimitType:    s.DialogLimitType,
		DialogLimitValue:   s.DialogLimitValue,
		UnlockCostDiamonds: s.UnlockCostDiamonds,
		Choices:            choiceResponses,
	}
}

// ProgressSummary is one row of a player's progress across all stories,
// joined with enough of the story/scene to render a library/continue-
// reading list without N+1 calls to GetOrCreateProgress (which has the
// side effect of creating a progress row for stories the player hasn't
// touched — unsuitable for a "what has this player already started"
// listing).
type ProgressSummary struct {
	Story             Story
	CurrentSceneID    uuid.UUID
	ChoicesMade       []uuid.UUID
	UpdatedAt         time.Time
	CurrentOrderIndex int32
	TotalScenes       int32
}

type ProgressSummaryResponse struct {
	Story                  StoryResponse `json:"story"`
	CurrentSceneID         uuid.UUID     `json:"current_scene_id"`
	ChoicesMade            []uuid.UUID   `json:"choices_made"`
	UpdatedAt              time.Time     `json:"updated_at"`
	CurrentSceneOrderIndex int32         `json:"current_scene_order_index"`
	TotalScenes            int32         `json:"total_scenes"`
	IsFinished             bool          `json:"is_finished"`
}

func toProgressSummaryResponse(p ProgressSummary) ProgressSummaryResponse {
	return ProgressSummaryResponse{
		Story:                  toStoryResponse(p.Story),
		CurrentSceneID:         p.CurrentSceneID,
		ChoicesMade:            p.ChoicesMade,
		UpdatedAt:              p.UpdatedAt,
		CurrentSceneOrderIndex: p.CurrentOrderIndex,
		TotalScenes:            p.TotalScenes,
		IsFinished:             p.CurrentOrderIndex >= p.TotalScenes-1,
	}
}

func toProgressSummaryResponses(summaries []ProgressSummary) []ProgressSummaryResponse {
	out := make([]ProgressSummaryResponse, 0, len(summaries))
	for _, s := range summaries {
		out = append(out, toProgressSummaryResponse(s))
	}
	return out
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
