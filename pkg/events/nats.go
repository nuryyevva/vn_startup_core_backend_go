// Package events wraps NATS JetStream connection setup and publishing so
// every module shares one stream and one JSON publish helper instead of
// talking to nats.go directly.
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// StreamName is the single JetStream stream backing every subject used by
// the platform's event contract (player.*, dialog.*, ai.*).
const StreamName = "VN_EVENTS"

// StreamSubjects lists the subject wildcards captured by StreamName.
var StreamSubjects = []string{"player.>", "dialog.>", "ai.>"}

// Subjects published by this service.
const (
	SubjectPlayerChoiceMade   = "player.choice.made"
	SubjectDialogSessionStart = "dialog.session.started"
	SubjectDialogMessageSent  = "dialog.message.sent"
	SubjectDialogSessionEnded = "dialog.session.ended"
)

// Subjects this service subscribes to, published by the AI Orchestrator.
const (
	SubjectAIResponseReady          = "ai.response.ready"
	SubjectAIDialogResponseChunk    = "ai.dialog.response.chunk"
	SubjectAIDialogResponseComplete = "ai.dialog.response.complete"
	SubjectAISessionFinalized       = "ai.session.finalized"
)

// Connect dials NATS and returns both the raw connection (closed by the
// caller on shutdown) and a JetStream context with StreamName ensured.
func Connect(url string) (*nats.Conn, jetstream.JetStream, error) {
	nc, err := nats.Connect(url, nats.Name("vn-core-backend"))
	if err != nil {
		return nil, nil, fmt.Errorf("connect to nats: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("create jetstream context: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     StreamName,
		Subjects: StreamSubjects,
	}); err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("ensure stream %s: %w", StreamName, err)
	}

	return nc, js, nil
}

// Publisher publishes JSON-encoded events to JetStream subjects.
type Publisher struct {
	js jetstream.JetStream
}

func NewPublisher(js jetstream.JetStream) *Publisher {
	return &Publisher{js: js}
}

// Publish marshals payload as JSON and publishes it to subject, waiting for
// the broker's ack.
func (p *Publisher) Publish(ctx context.Context, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event payload for %s: %w", subject, err)
	}

	if _, err := p.js.Publish(ctx, subject, data); err != nil {
		return fmt.Errorf("publish event to %s: %w", subject, err)
	}

	return nil
}
