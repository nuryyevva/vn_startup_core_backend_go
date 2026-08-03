package wallet

import (
	"github.com/gofiber/fiber/v2"

	"vn_startup_core_backend_go/pkg/apperr"
	"vn_startup_core_backend_go/pkg/middleware"
)

// DevHandler exposes the manual diamond-grant endpoint that stands in for a
// real payment provider on this stage of the project. RegisterRoutes must
// only be called when Config.Dev.EnableDevEndpoints is true; when disabled
// the route is simply never registered, so it 404s.
type DevHandler struct {
	service *Service
}

func NewDevHandler(service *Service) *DevHandler {
	return &DevHandler{service: service}
}

func (h *DevHandler) RegisterRoutes(router fiber.Router, auth fiber.Handler) {
	router.Post("/dev/wallet/grant", auth, h.grant)
}

func (h *DevHandler) grant(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	var req GrantRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.BadRequest("invalid_body", "Некорректное тело запроса")
	}

	tx, err := h.service.Grant(c.Context(), userID, req.Amount)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(toTransactionResponse(tx))
}
