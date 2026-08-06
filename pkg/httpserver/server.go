// Package httpserver builds the Fiber app shared by the whole service: a
// single JSON error format, request logging, and lifecycle management.
package httpserver

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"vn_startup_core_backend_go/pkg/apperr"
	"vn_startup_core_backend_go/pkg/middleware"
)

// New builds a Fiber app configured with the shared error handler, CORS,
// and request logging middleware. Route registration happens in
// cmd/api/main.go via the returned *fiber.App. allowedOrigins is a
// comma-separated origin list (or "*") — see config.CORSConfig for why the
// default is permissive.
func New(logger *slog.Logger, allowedOrigins string) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler:          errorHandler,
		DisableStartupMessage: true,
	})

	// Browser-only concern (mobile/desktop clients ignore CORS entirely):
	// without this, every request from the Flutter web dev server or a
	// deployed web build fails at the browser before it ever reaches this
	// handler, surfacing to the user as a generic "can't reach server"
	// network error rather than anything mentioning CORS.
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PATCH, PUT, DELETE, OPTIONS",
	}))
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
