// Package httpserver builds the Fiber app shared by the whole service: a
// single JSON error format, request logging, and lifecycle management.
package httpserver

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"vn_startup_core_backend_go/pkg/apperr"
	"vn_startup_core_backend_go/pkg/middleware"
)

// New builds a Fiber app configured with the shared error handler and
// request logging middleware. Route registration happens in cmd/api/main.go
// via the returned *fiber.App.
func New(logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler:          errorHandler,
		DisableStartupMessage: true,
	})

	app.Use(middleware.Logging(logger))

	return app
}

// errorHandler renders every error returned from a handler as
// {"error": {"code": "...", "message": "..."}}.
func errorHandler(c *fiber.Ctx, err error) error {
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return c.Status(appErr.Status).JSON(apperr.Envelope{Error: appErr})
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return c.Status(fiberErr.Code).JSON(apperr.Envelope{Error: &apperr.Error{
			Code:    "http_error",
			Message: fiberErr.Message,
		}})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(apperr.Envelope{Error: &apperr.Error{
		Code:    "internal_error",
		Message: "Внутренняя ошибка сервера",
	}})
}
