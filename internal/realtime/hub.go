// Package realtime delivers streamed AI responses and session notifications
// to connected clients over WebSocket.
package realtime

import (
	"fmt"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

// conn wraps a websocket connection with its own write mutex: the
// underlying gorilla/fasthttp websocket connection is not safe for
// concurrent writes, but both the NATS subscriber and the dialog scheduler
// may push to the same user concurrently.
type conn struct {
	mu sync.Mutex
	ws *websocket.Conn
}

func (c *conn) writeMessage(payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ws.WriteMessage(websocket.TextMessage, payload)
}

// Hub tracks one active WebSocket connection per authenticated user and
// delivers server-pushed JSON payloads to them.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]*conn
}

func NewHub() *Hub {
	return &Hub{conns: make(map[string]*conn)}
}

// Register associates userID with ws, replacing any previous connection for
// that user (a new tab/device reconnecting supersedes the old one).
func (h *Hub) Register(userID string, ws *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[userID] = &conn{ws: ws}
}

// Unregister removes userID's connection. Callers must pass the same *conn
// registered, checked by identity, so a stale Unregister after a
// reconnect doesn't evict the new connection.
func (h *Hub) Unregister(userID string, ws *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c, ok := h.conns[userID]; ok && c.ws == ws {
		delete(h.conns, userID)
	}
}

// SendToUser delivers payload to userID's active connection, if any.
func (h *Hub) SendToUser(userID string, payload []byte) error {
	h.mu.RLock()
	c, ok := h.conns[userID]
	h.mu.RUnlock()
	if !ok {
		return fmt.Errorf("realtime: no active connection for user %s", userID)
	}
	return c.writeMessage(payload)
}
