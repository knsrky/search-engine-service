// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Logger returns a middleware that logs HTTP requests.
func Logger(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Log request
		duration := time.Since(start)
		status := c.Response().StatusCode()

		fields := []zap.Field{
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.Duration("duration", duration),
			zap.String("ip", c.IP()),
			zap.String("user_agent", c.Get("User-Agent")),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
		}

		if status >= 500 {
			logger.Error("request failed", fields...)
		} else if status >= 400 {
			logger.Warn("request error", fields...)
		} else {
			logger.Debug("request completed", fields...)
		}

		return err
	}
}
