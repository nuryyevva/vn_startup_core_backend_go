package story

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

// RegisterRoutes wires the story endpoints onto router, gated by auth.
func (h *Handler) RegisterRoutes(router fiber.Router, auth fiber.Handler) {
	router.Get("/stories", auth, h.listStories)
	router.Get("/stories/:id/progress", auth, h.getProgress)
	router.Get("/scenes/:id", auth, h.getScene)
	router.Post("/scenes/:id/choice", auth, h.submitChoice)
}

func (h *Handler) listStories(c *fiber.Ctx) error {
	stories, err := h.service.ListPublishedStories(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(toStoryResponses(stories))
}

func (h *Handler) getProgress(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	storyID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.BadRequest("invalid_story_id", "Некорректный идентификатор истории")
	}

	progress, err := h.service.GetOrCreateProgress(c.Context(), userID, storyID)
	if err != nil {
		return err
	}

	return c.JSON(toProgressResponse(progress))
}

func (h *Handler) getScene(c *fiber.Ctx) error {
	sceneID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.BadRequest("invalid_scene_id", "Некорректный идентификатор сцены")
	}

	scene, choices, err := h.service.GetScene(c.Context(), sceneID)
	if err != nil {
		return err
	}

	return c.JSON(toSceneResponse(scene, choices))
}

func (h *Handler) submitChoice(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	sceneID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.BadRequest("invalid_scene_id", "Некорректный идентификатор сцены")
	}

	var req ChoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.BadRequest("invalid_body", "Некорректное тело запроса")
	}
	if req.ChoiceID == uuid.Nil {
		return apperr.BadRequest("missing_choice_id", "Не указан идентификатор выбора")
	}

	progress, err := h.service.SubmitChoice(c.Context(), userID, sceneID, req.ChoiceID)
	if err != nil {
		return err
	}

	return c.JSON(toProgressResponse(progress))
}
