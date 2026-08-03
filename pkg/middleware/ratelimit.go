package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"vn_startup_core_backend_go/pkg/apperr"
)

// RateLimitByIP returns a Fiber middleware enforcing a fixed-window rate
// limit of max requests per window, keyed by client IP and route, backed by
// Redis so the limit is shared across all instances of the service.
func RateLimitByIP(rdb *redis.Client, keyPrefix string, max int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		key := fmt.Sprintf("ratelimit:%s:%s", keyPrefix, c.IP())

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// Fail open: a Redis outage should not take down the API.
			return c.Next()
		}
		if count == 1 {
			rdb.Expire(ctx, key, window)
		}

		if count > int64(max) {
			return apperr.New(fiber.StatusTooManyRequests, "rate_limited", "Слишком много запросов, попробуйте позже")
		}

		return c.Next()
	}
}
