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

// RegisterRoutes wires the story endpoints onto router, gated by auth except
// for GET /stories: the catalog is public so guest mode (client-side only,
// no backend session) can still browse it, and listStories never reads
// user identity anyway.
func (h *Handler) RegisterRoutes(router fiber.Router, auth fiber.Handler) {
	router.Get("/stories", h.listStories)
	router.Get("/stories/:id", h.getStory)
	router.Get("/progress", auth, h.listMyProgress)
	router.Get("/stories/:id/progress", auth, h.getProgress)
	router.Get("/stories/:id/scenes", auth, h.listStoryScenes)
	router.Get("/scenes/:id", auth, h.getScene)
	router.Post("/scenes/:id/choice", auth, h.submitChoice)
	router.Post("/scenes/:id/unlock", auth, h.unlockScene)
	router.Get("/bookmarks", auth, h.listBookmarks)
	router.Post("/stories/:id/bookmark", auth, h.addBookmark)
	router.Delete("/stories/:id/bookmark", auth, h.removeBookmark)
}

func (h *Handler) listStories(c *fiber.Ctx) error {
	var genre *string
	if g := c.Query("genre"); g != "" {
		genre = &g
	}
	page := int32(c.QueryInt("page", 1))
	pageSize := int32(c.QueryInt("page_size", 0))

	result, err := h.service.ListPublishedStories(c.Context(), genre, page, pageSize)
	if err != nil {
		return err
	}
	return c.JSON(toStoryListResponse(result))
}

func (h *Handler) getStory(c *fiber.Ctx) error {
	storyID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.BadRequest("invalid_story_id", "Некорректный идентификатор истории")
	}

	story, err := h.service.GetPublishedStory(c.Context(), storyID)
	if err != nil {
		return err
	}
	return c.JSON(toStoryResponse(story))
}

func (h *Handler) listBookmarks(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	stories, err := h.service.ListBookmarks(c.Context(), userID)
	if err != nil {
		return err
	}
	return c.JSON(toStoryResponses(stories))
}

func (h *Handler) addBookmark(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	storyID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.BadRequest("invalid_story_id", "Некорректный идентификатор истории")
	}

	if err := h.service.Bookmark(c.Context(), userID, storyID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) removeBookmark(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	storyID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.BadRequest("invalid_story_id", "Некорректный идентификатор истории")
	}

	if err := h.service.Unbookmark(c.Context(), userID, storyID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) listMyProgress(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	summaries, err := h.service.ListMyProgress(c.Context(), userID)
	if err != nil {
		return err
	}

	return c.JSON(toProgressSummaryResponses(summaries))
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
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	sceneID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.BadRequest("invalid_scene_id", "Некорректный идентификатор сцены")
	}

	scene, choices, err := h.service.GetSceneForPlayer(c.Context(), userID, sceneID)
	if err != nil {
		return err
	}

	return c.JSON(toSceneResponse(scene, choices))
}

func (h *Handler) listStoryScenes(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	storyID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.BadRequest("invalid_story_id", "Некорректный идентификатор истории")
	}

	scenes, unlockedSceneIDs, err := h.service.ListStoryScenes(c.Context(), userID, storyID)
	if err != nil {
		return err
	}

	return c.JSON(toSceneSummaryResponses(scenes, unlockedSceneIDs))
}

func (h *Handler) unlockScene(c *fiber.Ctx) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return err
	}

	sceneID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.BadRequest("invalid_scene_id", "Некорректный идентификатор сцены")
	}

	if err := h.service.UnlockScene(c.Context(), userID, sceneID); err != nil {
		return err
	}

	return c.JSON(SceneUnlockResponse{SceneID: sceneID, Unlocked: true})
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
