package middleware

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"vn_startup_core_backend_go/internal/auth"
	"vn_startup_core_backend_go/pkg/apperr"
)

// LocalsUserID and LocalsRole re-export auth.ContextUserIDKey/ContextRoleKey
// under this package's own name for callers that only import pkg/middleware.
const (
	LocalsUserID = auth.ContextUserIDKey
	LocalsRole   = auth.ContextRoleKey
)

// Auth returns a Fiber middleware that requires a valid access-token JWT,
// either as a Bearer Authorization header or as a `token` query parameter
// (needed for the WebSocket handshake, which cannot set headers from a
// browser EventSource/WebSocket client).
func Auth(issuer *auth.TokenIssuer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			return apperr.Unauthorized("missing_token", "Отсутствует токен авторизации")
		}

		claims, err := issuer.ParseAccessToken(tokenStr)
		if err != nil {
			if errors.Is(err, auth.ErrExpiredToken) {
				return apperr.Unauthorized("token_expired", "Токен авторизации истёк")
			}
			return apperr.Unauthorized("invalid_token", "Недействительный токен авторизации")
		}

		c.Locals(LocalsUserID, claims.UserID)
		c.Locals(LocalsRole, claims.Role)
		return c.Next()
	}
}

// UserIDFromContext reads the authenticated user's ID set by Auth. It
// returns an apperr.Error if called on a route not protected by Auth.
func UserIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	userID, ok := c.Locals(LocalsUserID).(uuid.UUID)
	if !ok {
		return uuid.Nil, apperr.Unauthorized("missing_token", "Отсутствует токен авторизации")
	}
	return userID, nil
}

func extractToken(c *fiber.Ctx) string {
	header := c.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	if q := c.Query("token"); q != "" {
		return q
	}
	return ""
}
