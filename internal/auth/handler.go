package auth

import (
	"github.com/gofiber/fiber/v2"

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
