package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"vn_startup_core_backend_go/pkg/apperr"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes wires /auth/register, /auth/login, /auth/refresh onto router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/auth/register", h.register)
	router.Post("/auth/login", h.login)
	router.Post("/auth/refresh", h.refresh)
}

// RegisterProtectedRoutes wires /auth/change-password, gated by auth (unlike
// register/login/refresh, which must work without an existing session).
func (h *Handler) RegisterProtectedRoutes(router fiber.Router, auth fiber.Handler) {
	router.Post("/auth/change-password", auth, h.changePassword)
}

func (h *Handler) register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.BadRequest("invalid_body", "Некорректное тело запроса")
	}

	user, tokens, err := h.service.Register(c.Context(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(AuthResponse{
		User:   toUserResponse(user),
		Tokens: tokens,
	})
}

func (h *Handler) login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.BadRequest("invalid_body", "Некорректное тело запроса")
	}

	user, tokens, err := h.service.Login(c.Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(AuthResponse{
		User:   toUserResponse(user),
		Tokens: tokens,
	})
}

func (h *Handler) refresh(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.BadRequest("invalid_body", "Некорректное тело запроса")
	}
	if req.RefreshToken == "" {
		return apperr.BadRequest("missing_refresh_token", "Отсутствует refresh-токен")
	}

	tokens, err := h.service.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return err
	}

	return c.JSON(tokens)
}

func (h *Handler) changePassword(c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return err
	}

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.BadRequest("invalid_body", "Некорректное тело запроса")
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		return apperr.BadRequest("missing_fields", "Не указан текущий или новый пароль")
	}

	if err := h.service.ChangePassword(c.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// userIDFromContext reads the authenticated user's ID set by
// pkg/middleware.Auth. It duplicates pkg/middleware.UserIDFromContext's
// logic rather than importing that package, which would create an import
// cycle (pkg/middleware already imports this package).
func userIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	userID, ok := c.Locals(ContextUserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, apperr.Unauthorized("missing_token", "Отсутствует токен авторизации")
	}
	return userID, nil
}
