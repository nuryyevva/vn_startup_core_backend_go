package realtime

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"

	"vn_startup_core_backend_go/internal/dialog"
	"vn_startup_core_backend_go/pkg/events"
)

// DialogRecorder is the subset of dialog.Service the subscriber needs: to
// persist AI-generated replies once they're complete, and to resolve which
// user owns a session so chunk/complete events (which carry only a
// session_id) can be routed to the right WebSocket connection.
type DialogRecorder interface {
	RecordCharacterMessage(ctx context.Context, sessionID uuid.UUID, text string) (dialog.Message, error)
	SessionOwner(ctx context.Context, sessionID uuid.UUID) (uuid.UUID, error)
}

// Subscriber consumes the AI Orchestrator's events (ai.>) from JetStream and
// fans them out to the WebSocket hub, persisting completed dialog replies
// along the way. Core Backend only subscribes to these subjects; it never
// publishes them.
type Subscriber struct {
	js       jetstream.JetStream
	hub      *Hub
	recorder DialogRecorder
	logger   *slog.Logger
}

func NewSubscriber(js jetstream.JetStream, hub *Hub, recorder DialogRecorder, logger *slog.Logger) *Subscriber {
	if logger == nil {
		logger = slog.Default()
	}
	return &Subscriber{js: js, hub: hub, recorder: recorder, logger: logger}
}

// Start creates (or reuses) a durable pull consumer on the shared events
// stream filtered to ai.> subjects and begins processing messages
// asynchronously. It returns once the consumer is set up; processing
// continues in the background until ctx is cancelled.
func (s *Subscriber) Start(ctx context.Context) error {
	consumer, err := s.js.CreateOrUpdateConsumer(ctx, events.StreamName, jetstream.ConsumerConfig{
		Durable: "core_backend_ai_events",
		FilterSubjects: []string{
			events.SubjectAIResponseReady,
			events.SubjectAIDialogResponseChunk,
			events.SubjectAIDialogResponseComplete,
			events.SubjectAISessionFinalized,
		},
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return err
	}

	consumeCtx, err := consumer.Consume(s.handleMessage)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		consumeCtx.Stop()
	}()

	return nil
}

func (s *Subscriber) handleMessage(msg jetstream.Msg) {
	ctx := context.Background()

	switch msg.Subject() {
	case events.SubjectAIResponseReady:
		s.handleResponseReady(msg)
	case events.SubjectAIDialogResponseChunk:
		s.handleResponseChunk(msg)
	case events.SubjectAIDialogResponseComplete:
		s.handleResponseComplete(ctx, msg)
	case events.SubjectAISessionFinalized:
		s.handleSessionFinalized(msg)
	default:
		s.logger.Warn("realtime: unhandled subject", slog.String("subject", msg.Subject()))
	}

	if err := msg.Ack(); err != nil {
		s.logger.Error("realtime: ack failed", slog.String("subject", msg.Subject()), slog.String("error", err.Error()))
	}
}

type aiResponseReadyPayload struct {
	UserID       uuid.UUID `json:"user_id"`
	SceneID      uuid.UUID `json:"scene_id"`
	ResponseText string    `json:"response_text"`
}

type wsAIResponseReady struct {
	Type    string    `json:"type"`
	SceneID uuid.UUID `json:"scene_id"`
	Text    string    `json:"text"`
}

func (s *Subscriber) handleResponseReady(msg jetstream.Msg) {
	var payload aiResponseReadyPayload
	if !s.unmarshal(msg, &payload) {
		return
	}

	s.send(payload.UserID, wsAIResponseReady{
		Type:    "ai_response_ready",
		SceneID: payload.SceneID,
		Text:    payload.ResponseText,
	})
}

type aiDialogResponseChunkPayload struct {
	SessionID uuid.UUID `json:"session_id"`
	MessageID uuid.UUID `json:"message_id"`
	ChunkText string    `json:"chunk_text"`
}

type wsAIResponseChunk struct {
	Type      string    `json:"type"`
	SessionID uuid.UUID `json:"session_id"`
	Chunk     string    `json:"chunk"`
}

func (s *Subscriber) handleResponseChunk(msg jetstream.Msg) {
	var payload aiDialogResponseChunkPayload
	if !s.unmarshal(msg, &payload) {
		return
	}

	s.broadcastBySession(payload.SessionID, wsAIResponseChunk{
		Type:      "ai_response_chunk",
		SessionID: payload.SessionID,
		Chunk:     payload.ChunkText,
	})
}

type aiDialogResponseCompletePayload struct {
	SessionID uuid.UUID `json:"session_id"`
	MessageID uuid.UUID `json:"message_id"`
	FullText  string    `json:"full_text"`
}

type wsAIResponseComplete struct {
	Type      string    `json:"type"`
	SessionID uuid.UUID `json:"session_id"`
	Text      string    `json:"text"`
}

func (s *Subscriber) handleResponseComplete(ctx context.Context, msg jetstream.Msg) {
	var payload aiDialogResponseCompletePayload
	if !s.unmarshal(msg, &payload) {
		return
	}

	if _, err := s.recorder.RecordCharacterMessage(ctx, payload.SessionID, payload.FullText); err != nil {
		s.logger.Error("realtime: persist character message failed",
			slog.String("session_id", payload.SessionID.String()), slog.String("error", err.Error()))
	}

	s.broadcastBySession(payload.SessionID, wsAIResponseComplete{
		Type:      "ai_response_complete",
		SessionID: payload.SessionID,
		Text:      payload.FullText,
	})
}

type aiSessionFinalizedPayload struct {
	SessionID uuid.UUID `json:"session_id"`
}

func (s *Subscriber) handleSessionFinalized(msg jetstream.Msg) {
	var payload aiSessionFinalizedPayload
	if !s.unmarshal(msg, &payload) {
		return
	}
	s.logger.Info("realtime: ai session finalized", slog.String("session_id", payload.SessionID.String()))
}

func (s *Subscriber) unmarshal(msg jetstream.Msg, v any) bool {
	if err := json.Unmarshal(msg.Data(), v); err != nil {
		s.logger.Error("realtime: unmarshal event failed", slog.String("subject", msg.Subject()), slog.String("error", err.Error()))
		return false
	}
	return true
}

func (s *Subscriber) send(userID uuid.UUID, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("realtime: marshal ws payload failed", slog.String("error", err.Error()))
		return
	}
	if err := s.hub.SendToUser(userID.String(), data); err != nil {
		s.logger.Warn("realtime: send to user failed", slog.String("user_id", userID.String()), slog.String("error", err.Error()))
	}
}

// broadcastBySession resolves sessionID's owner (the AI Orchestrator's
// chunk/complete events carry only a session_id, and the hub is keyed by
// user) and delivers payload to that user's connection.
func (s *Subscriber) broadcastBySession(sessionID uuid.UUID, payload any) {
	ctx := context.Background()
	userID, err := s.recorder.SessionOwner(ctx, sessionID)
	if err != nil {
		s.logger.Error("realtime: resolve session owner failed", slog.String("session_id", sessionID.String()), slog.String("error", err.Error()))
		return
	}
	s.send(userID, payload)
}
