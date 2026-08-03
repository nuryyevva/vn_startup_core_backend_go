// Command mockai is a throwaway stand-in for the real Python AI
// Orchestrator, used only to exercise the WebSocket path end-to-end without
// standing up the actual LLM service. It subscribes to the events Core
// Backend publishes (player.choice.made, dialog.session.started,
// dialog.message.sent, dialog.session.ended) and replies with the ai.*
// events the real orchestrator is expected to publish, using canned text
// and artificial delays to emulate LLM latency and token streaming.
//
// It is not part of the production service and must never be started
// alongside a real AI Orchestrator on the same NATS deployment.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"

	"vn_startup_core_backend_go/internal/config"
	"vn_startup_core_backend_go/pkg/events"
	"vn_startup_core_backend_go/pkg/logger"
)

// mockaiConsumerDurable must differ from internal/realtime.Subscriber's
// durable name ("core_backend_ai_events"): that consumer filters on ai.>
// subjects while this one filters on player.>/dialog.> subjects, so they
// never overlap, but a distinct, clearly-named durable also makes it
// obvious in `nats consumer ls` which process owns which consumer.
const mockaiConsumerDurable = "mockai_consumer"

func main() {
	if err := run(); err != nil {
		slog.Error("mockai: fatal startup error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.Server.Env)

	nc, js, err := events.Connect(cfg.NATS.URL)
	if err != nil {
		return fmt.Errorf("connect nats: %w", err)
	}
	defer nc.Close()

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), events.StreamName, jetstream.ConsumerConfig{
		Durable: mockaiConsumerDurable,
		FilterSubjects: []string{
			events.SubjectPlayerChoiceMade,
			events.SubjectDialogSessionStart,
			events.SubjectDialogMessageSent,
			events.SubjectDialogSessionEnded,
		},
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	mock := &mockAI{publisher: events.NewPublisher(js), logger: log}

	consumeCtx, err := consumer.Consume(mock.handle)
	if err != nil {
		return fmt.Errorf("start consuming: %w", err)
	}
	defer consumeCtx.Stop()

	log.Info("mockai: listening", slog.String("nats_url", cfg.NATS.URL), slog.String("durable", mockaiConsumerDurable))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Info("mockai: shutting down")
	return nil
}

type mockAI struct {
	publisher *events.Publisher
	logger    *slog.Logger
}

// handle dispatches one JetStream message by subject. Messages are acked
// immediately since this is a mock: there's nothing worth redelivering, and
// blocking the whole consumer on the artificial LLM-latency sleeps below
// would serialize unrelated events for no reason, so the actual work runs
// in its own goroutine.
func (m *mockAI) handle(msg jetstream.Msg) {
	subject := msg.Subject()
	data := msg.Data()

	if err := msg.Ack(); err != nil {
		m.logger.Error("mockai: ack failed", slog.String("subject", subject), slog.String("error", err.Error()))
	}

	go func() {
		switch subject {
		case events.SubjectPlayerChoiceMade:
			m.handlePlayerChoiceMade(data)
		case events.SubjectDialogSessionStart:
			m.handleDialogSessionStarted(data)
		case events.SubjectDialogMessageSent:
			m.handleDialogMessageSent(data)
		case events.SubjectDialogSessionEnded:
			m.handleDialogSessionEnded(data)
		default:
			m.logger.Warn("mockai: unhandled subject", slog.String("subject", subject))
		}
	}()
}

type playerChoiceMadeEvent struct {
	UserID   uuid.UUID `json:"user_id"`
	StoryID  uuid.UUID `json:"story_id"`
	SceneID  uuid.UUID `json:"scene_id"`
	ChoiceID uuid.UUID `json:"choice_id"`
}

type aiResponseReadyPayload struct {
	UserID       uuid.UUID `json:"user_id"`
	SceneID      uuid.UUID `json:"scene_id"`
	ResponseText string    `json:"response_text"`
}

// responseReadyPhrases stand in for a scene-level LLM response to a
// player's choice; picked at random so repeated manual tests don't look
// stuck on one canned line.
var responseReadyPhrases = []string{
	"Тень скользнула по стене, и в комнате стало тише, чем прежде. Ты чувствуешь, что этот выбор изменит куда больше, чем кажется на первый взгляд.",
	"Голос за спиной произносит твоё имя так, будто знал его целую вечность. Что-то внутри тебя дрогнуло, и пути назад уже нет.",
	"Свеча на столе внезапно гаснет, оставляя после себя лишь тонкую струйку дыма. Решение принято — и мир вокруг тебя уже никогда не будет прежним.",
}

func (m *mockAI) handlePlayerChoiceMade(data []byte) {
	var evt playerChoiceMadeEvent
	if !m.unmarshal(events.SubjectPlayerChoiceMade, data, &evt) {
		return
	}
	m.logger.Info("mockai: received player.choice.made", slog.Any("event", evt))

	time.Sleep(randomDuration(1*time.Second, 2*time.Second))

	m.publish(events.SubjectAIResponseReady, aiResponseReadyPayload{
		UserID:       evt.UserID,
		SceneID:      evt.SceneID,
		ResponseText: pick(responseReadyPhrases),
	})
}

type dialogSessionStartedEvent struct {
	SessionID   uuid.UUID `json:"session_id"`
	UserID      uuid.UUID `json:"user_id"`
	CharacterID uuid.UUID `json:"character_id"`
	SceneID     uuid.UUID `json:"scene_id"`
}

func (m *mockAI) handleDialogSessionStarted(data []byte) {
	var evt dialogSessionStartedEvent
	if !m.unmarshal(events.SubjectDialogSessionStart, data, &evt) {
		return
	}
	// A real orchestrator would open a LangGraph thread for this session
	// here. The mock has no state to initialize, so it just logs receipt.
	m.logger.Info("mockai: received dialog.session.started (no-op)", slog.Any("event", evt))
}

type dialogMessageSentEvent struct {
	SessionID      uuid.UUID `json:"session_id"`
	MessageID      uuid.UUID `json:"message_id"`
	UserID         uuid.UUID `json:"user_id"`
	CharacterID    uuid.UUID `json:"character_id"`
	Text           string    `json:"text"`
	RemainingLimit *int32    `json:"remaining_limit"`
}

type aiDialogResponseChunkPayload struct {
	SessionID uuid.UUID `json:"session_id"`
	MessageID uuid.UUID `json:"message_id"`
	ChunkText string    `json:"chunk_text"`
}

type aiDialogResponseCompletePayload struct {
	SessionID uuid.UUID `json:"session_id"`
	MessageID uuid.UUID `json:"message_id"`
	FullText  string    `json:"full_text"`
}

// dialogReplyPhrases stand in for a character's free-dialog reply; each is
// long enough to split into 4-6 word-chunks for the fake streaming below.
var dialogReplyPhrases = []string{
	"Ты правда думаешь, что это было правильным решением? Иногда цена оказывается выше, чем кажется на первый взгляд. Но что бы ни случилось, я останусь рядом.",
	"Знаешь, я давно ждал этого разговора. Есть вещи, которые я не решался тебе рассказать раньше. Возможно, сейчас как раз подходящее время для этого.",
	"Между нами всегда было что-то невысказанное. Сегодня звёзды как никогда ясны, и мне кажется, это неспроста. Останься ещё немного, прошу тебя.",
	"Каждый твой шаг здесь имеет значение, даже если сейчас это неочевидно. Помни: эта история пишется не только словами, но и молчанием между ними.",
}

func (m *mockAI) handleDialogMessageSent(data []byte) {
	var evt dialogMessageSentEvent
	if !m.unmarshal(events.SubjectDialogMessageSent, data, &evt) {
		return
	}
	m.logger.Info("mockai: received dialog.message.sent", slog.Any("event", evt))

	fullText := pick(dialogReplyPhrases)
	messageID := uuid.New()

	for _, chunk := range splitIntoChunks(fullText, 4, 6) {
		time.Sleep(randomDuration(200*time.Millisecond, 400*time.Millisecond))
		m.publish(events.SubjectAIDialogResponseChunk, aiDialogResponseChunkPayload{
			SessionID: evt.SessionID,
			MessageID: messageID,
			ChunkText: chunk,
		})
	}

	m.publish(events.SubjectAIDialogResponseComplete, aiDialogResponseCompletePayload{
		SessionID: evt.SessionID,
		MessageID: messageID,
		FullText:  fullText,
	})
}

type dialogSessionEndedEvent struct {
	SessionID uuid.UUID `json:"session_id"`
	Reason    string    `json:"reason"`
}

type aiSessionFinalizedPayload struct {
	SessionID uuid.UUID `json:"session_id"`
}

func (m *mockAI) handleDialogSessionEnded(data []byte) {
	var evt dialogSessionEndedEvent
	if !m.unmarshal(events.SubjectDialogSessionEnded, data, &evt) {
		return
	}
	m.logger.Info("mockai: received dialog.session.ended", slog.Any("event", evt))

	time.Sleep(randomDuration(300*time.Millisecond, 500*time.Millisecond))

	m.publish(events.SubjectAISessionFinalized, aiSessionFinalizedPayload{SessionID: evt.SessionID})
}

func (m *mockAI) unmarshal(subject string, data []byte, v any) bool {
	if err := json.Unmarshal(data, v); err != nil {
		m.logger.Error("mockai: unmarshal event failed", slog.String("subject", subject), slog.String("error", err.Error()))
		return false
	}
	return true
}

func (m *mockAI) publish(subject string, payload any) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.publisher.Publish(ctx, subject, payload); err != nil {
		m.logger.Error("mockai: publish failed", slog.String("subject", subject), slog.String("error", err.Error()))
		return
	}
	m.logger.Info("mockai: published event", slog.String("subject", subject), slog.Any("payload", payload))
}

// splitIntoChunks breaks text into somewhere between minChunks and
// maxChunks word groups, emulating how a real LLM streams a reply as
// several token chunks rather than one shot.
func splitIntoChunks(text string, minChunks, maxChunks int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	n := minChunks + rand.Intn(maxChunks-minChunks+1)
	if n > len(words) {
		n = len(words)
	}
	if n < 1 {
		n = 1
	}

	chunks := make([]string, 0, n)
	perChunk := len(words) / n
	remainder := len(words) % n
	idx := 0
	for i := 0; i < n; i++ {
		size := perChunk
		if i < remainder {
			size++
		}
		if size == 0 {
			continue
		}
		chunks = append(chunks, strings.Join(words[idx:idx+size], " "))
		idx += size
	}
	return chunks
}

func randomDuration(minDur, maxDur time.Duration) time.Duration {
	if maxDur <= minDur {
		return minDur
	}
	return minDur + time.Duration(rand.Int63n(int64(maxDur-minDur)))
}

func pick(phrases []string) string {
	return phrases[rand.Intn(len(phrases))]
}
