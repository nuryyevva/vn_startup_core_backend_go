package achievement

import (
	"github.com/gofiber/fiber/v2"

	"vn_startup_core_backend_go/pkg/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes wires GET /achievements, gated by auth.
func (h *Handler) RegisterRoutes(router fiber.Router, auth fiber.Handler) {
	router.Get("/achievements", auth, h.listAchievements)
}

func (h *Handler) listAchievements(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	achievements, err := h.service.ListAchievements(c.Context(), userID)
	if err != nil {
		return err
	}
	return c.JSON(toAchievementResponses(achievements))
}
