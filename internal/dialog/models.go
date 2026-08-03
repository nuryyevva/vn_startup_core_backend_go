package dialog

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive      = "active"
	StatusEnded       = "ended"
	StatusInterrupted = "interrupted"

	LimitTypeTime     = "time"
	LimitTypeMessages = "messages"
	LimitTypeNone     = "none"

	SenderUser      = "user"
	SenderCharacter = "character"

	EndReasonUser         = "user_ended"
	EndReasonTimeLimit    = "time_limit"
	EndReasonMessageLimit = "message_limit"
)

type Session struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	CharacterID  uuid.UUID
	SceneID      uuid.UUID
	Status       string
	LimitType    string
	LimitValue   *int32
	MessageCount int32
	StartedAt    time.Time
	EndedAt      *time.Time
	EndReason    *string
}

type Message struct {
	ID           uuid.UUID
	SessionID    uuid.UUID
	Sender       string
	Text         string
	CostDiamonds int32
	CreatedAt    time.Time
}

type StartSessionRequest struct {
	CharacterID uuid.UUID `json:"character_id"`
	SceneID     uuid.UUID `json:"scene_id"`
}

type SendMessageRequest struct {
	Text string `json:"text"`
}

type SessionResponse struct {
	ID           uuid.UUID  `json:"id"`
	CharacterID  uuid.UUID  `json:"character_id"`
	SceneID      uuid.UUID  `json:"scene_id"`
	Status       string     `json:"status"`
	LimitType    string     `json:"limit_type"`
	LimitValue   *int32     `json:"limit_value,omitempty"`
	MessageCount int32      `json:"message_count"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	EndReason    *string    `json:"end_reason,omitempty"`
}

func toSessionResponse(s Session) SessionResponse {
	return SessionResponse{
		ID:           s.ID,
		CharacterID:  s.CharacterID,
		SceneID:      s.SceneID,
		Status:       s.Status,
		LimitType:    s.LimitType,
		LimitValue:   s.LimitValue,
		MessageCount: s.MessageCount,
		StartedAt:    s.StartedAt,
		EndedAt:      s.EndedAt,
		EndReason:    s.EndReason,
	}
}

type MessageResponse struct {
	ID             uuid.UUID `json:"id"`
	SessionID      uuid.UUID `json:"session_id"`
	Sender         string    `json:"sender"`
	Text           string    `json:"text"`
	CostDiamonds   int32     `json:"cost_diamonds"`
	CreatedAt      time.Time `json:"created_at"`
	RemainingLimit *int32    `json:"remaining_limit,omitempty"`
}

func toMessageResponse(m Message, remaining *int32) MessageResponse {
	return MessageResponse{
		ID:             m.ID,
		SessionID:      m.SessionID,
		Sender:         m.Sender,
		Text:           m.Text,
		CostDiamonds:   m.CostDiamonds,
		CreatedAt:      m.CreatedAt,
		RemainingLimit: remaining,
	}
}
