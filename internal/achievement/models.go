package achievement

import "time"

// Achievement is one catalog entry joined with userID's unlock state.
type Achievement struct {
	ID          string
	Title       string
	Description string
	Unlocked    bool
	UnlockedAt  *time.Time
}

type AchievementResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Unlocked    bool       `json:"unlocked"`
	UnlockedAt  *time.Time `json:"unlocked_at,omitempty"`
}

func toAchievementResponse(a Achievement) AchievementResponse {
	return AchievementResponse(a)
}

func toAchievementResponses(achievements []Achievement) []AchievementResponse {
	out := make([]AchievementResponse, 0, len(achievements))
	for _, a := range achievements {
		out = append(out, toAchievementResponse(a))
	}
	return out
}
