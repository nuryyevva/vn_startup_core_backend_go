package user

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

// RegisterRoutes wires GET/PATCH /users/me, gated by auth.
func (h *Handler) RegisterRoutes(router fiber.Router, auth fiber.Handler) {
	router.Get("/users/me", auth, h.getMe)
	router.Patch("/users/me", auth, h.updateMe)
}

func (h *Handler) getMe(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	profile, err := h.service.GetProfile(c.Context(), userID)
	if err != nil {
		return err
	}

	return c.JSON(toProfileResponse(profile))
}

func (h *Handler) updateMe(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.BadRequest("invalid_body", "Некорректное тело запроса")
	}

	profile, err := h.service.UpdateProfile(c.Context(), userID, req)
	if err != nil {
		return err
	}

	return c.JSON(toProfileResponse(profile))
}
