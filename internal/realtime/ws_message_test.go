package realtime

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWSAIResponseComplete_EmotionAndSpriteAreOptional(t *testing.T) {
	id := uuid.New()

	plain, err := json.Marshal(wsAIResponseComplete{Type: "ai_response_complete", SessionID: id, Text: "Привет"})
	require.NoError(t, err)
	assert.NotContains(t, string(plain), "emotion")
	assert.NotContains(t, string(plain), "sprite_url")

	styled, err := json.Marshal(wsAIResponseComplete{
		Type: "ai_response_complete", SessionID: id, Text: "Привет", Emotion: "sad", SpriteURL: "/media/x.webp",
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"ai_response_complete","session_id":"`+id.String()+`","text":"Привет","emotion":"sad","sprite_url":"/media/x.webp"}`, string(styled))
}

func TestAIDialogResponseCompletePayload_ReadsEmotionAndSprite(t *testing.T) {
	var payload aiDialogResponseCompletePayload
	raw := `{"session_id":"` + uuid.New().String() + `","message_id":"` + uuid.New().String() +
		`","full_text":"Ну привет.","emotion":"happy","sprite_url":"/media/y.webp"}`
	require.NoError(t, json.Unmarshal([]byte(raw), &payload))
	assert.Equal(t, "happy", payload.Emotion)
	assert.Equal(t, "/media/y.webp", payload.SpriteURL)

	var legacy aiDialogResponseCompletePayload
	require.NoError(t, json.Unmarshal([]byte(`{"full_text":"Старый формат"}`), &legacy))
	assert.Empty(t, legacy.Emotion)
}
