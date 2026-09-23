package stats

import (
	"github.com/gofiber/fiber/v2"

	"vn_startup_core_backend_go/pkg/apperr"
	"vn_startup_core_backend_go/pkg/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes wires GET /stats/me and POST /stats/heartbeat, both gated
// by auth.
func (h *Handler) RegisterRoutes(router fiber.Router, auth fiber.Handler) {
	router.Get("/stats/me", auth, h.getStats)
	router.Post("/stats/heartbeat", auth, h.heartbeat)
}

func (h *Handler) getStats(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	stats, err := h.service.GetStats(c.Context(), userID)
	if err != nil {
		return err
	}
	return c.JSON(toStatsResponse(stats))
}

func (h *Handler) heartbeat(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	var req HeartbeatRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.BadRequest("invalid_body", "Некорректное тело запроса")
	}

	stats, err := h.service.Heartbeat(c.Context(), userID, req.Seconds)
	if err != nil {
		return err
	}
	return c.JSON(toStatsResponse(stats))
}
