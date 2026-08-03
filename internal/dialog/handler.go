package dialog

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"vn_startup_core_backend_go/pkg/apperr"
	"vn_startup_core_backend_go/pkg/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes wires the dialog endpoints onto router, gated by auth.
func (h *Handler) RegisterRoutes(router fiber.Router, auth fiber.Handler) {
	router.Post("/dialog/sessions", auth, h.startSession)
	router.Post("/dialog/sessions/:id/messages", auth, h.sendMessage)
	router.Post("/dialog/sessions/:id/end", auth, h.endSession)
}

func (h *Handler) startSession(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	var req StartSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.BadRequest("invalid_body", "Некорректное тело запроса")
	}
	if req.CharacterID == uuid.Nil || req.SceneID == uuid.Nil {
		return apperr.BadRequest("missing_fields", "Не указаны character_id или scene_id")
	}

	session, err := h.service.StartSession(c.Context(), userID, req.CharacterID, req.SceneID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(toSessionResponse(session))
}

func (h *Handler) sendMessage(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.BadRequest("invalid_session_id", "Некорректный идентификатор сессии")
	}

	var req SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.BadRequest("invalid_body", "Некорректное тело запроса")
	}
	if req.Text == "" {
		return apperr.BadRequest("empty_message", "Сообщение не может быть пустым")
	}

	message, remaining, err := h.service.SendMessage(c.Context(), userID, sessionID, req.Text)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(toMessageResponse(message, remaining))
}

func (h *Handler) endSession(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.BadRequest("invalid_session_id", "Некорректный идентификатор сессии")
	}

	session, err := h.service.EndSession(c.Context(), userID, sessionID)
	if err != nil {
		return err
	}

	return c.JSON(toSessionResponse(session))
}
