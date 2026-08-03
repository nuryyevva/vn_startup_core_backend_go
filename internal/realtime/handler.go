package realtime

import (
	"log/slog"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"vn_startup_core_backend_go/pkg/middleware"
)

type Handler struct {
	hub    *Hub
	logger *slog.Logger
}

func NewHandler(hub *Hub, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{hub: hub, logger: logger}
}

// RegisterRoutes wires GET /ws. issuer-based auth happens via
// middleware.Auth, which also accepts the token as a `token` query
// parameter since browser WebSocket clients can't set an Authorization
// header during the handshake.
func (h *Handler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	router.Get("/ws", authMiddleware, websocket.New(h.serve))
}

func (h *Handler) serve(c *websocket.Conn) {
	userID, ok := c.Locals(middleware.LocalsUserID).(uuid.UUID)
	if !ok {
		_ = c.Close()
		return
	}
	userIDStr := userID.String()

	h.hub.Register(userIDStr, c)
	defer h.hub.Unregister(userIDStr, c)

	for {
		if _, _, err := c.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				h.logger.Warn("websocket read error", slog.String("user_id", userIDStr), slog.String("error", err.Error()))
			}
			return
		}
		// This service only pushes server->client notifications; any
		// client-sent frame (e.g. ping) is read and discarded to keep the
		// connection alive and detect disconnects.
	}
}
