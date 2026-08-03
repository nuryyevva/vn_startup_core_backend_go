package wallet

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

// RegisterRoutes wires GET /wallet/balance, gated by auth.
func (h *Handler) RegisterRoutes(router fiber.Router, auth fiber.Handler) {
	router.Get("/wallet/balance", auth, h.getBalance)
}

func (h *Handler) getBalance(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	balance, err := h.service.Balance(c.Context(), userID)
	if err != nil {
		return err
	}

	return c.JSON(BalanceResponse{Balance: balance})
}
