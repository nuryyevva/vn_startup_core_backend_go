package wallet

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"vn_startup_core_backend_go/pkg/apperr"
	"vn_startup_core_backend_go/pkg/middleware"
)

// TopUpHandler exposes the manual diamond top-up that stands in for a real
// payment provider at this stage of the project. Only admins may use it;
// for everyone else buying diamonds is "coming soon", and the endpoint says
// exactly that (403 top_up_unavailable), which the client shows as is.
type TopUpHandler struct {
	service *Service
}

func NewTopUpHandler(service *Service) *TopUpHandler {
	return &TopUpHandler{service: service}
}

func (h *TopUpHandler) RegisterRoutes(router fiber.Router, auth fiber.Handler) {
	router.Post("/dev/wallet/grant", auth, h.grant)
}

func (h *TopUpHandler) grant(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}
	if middleware.RoleFromContext(c) != RoleAdmin {
		return apperr.New(http.StatusForbidden, "top_up_unavailable", "Пополнение скоро будет доступно")
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

// RoleAdmin is the users.role value allowed to top wallets up.
const RoleAdmin = "admin"
