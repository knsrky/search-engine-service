package middleware

import (
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"search-engine-service/internal/transport/httpserver/dto"
)

// Recover returns a middleware that recovers from panics.
func Recover(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					zap.Any("error", r),
					zap.String("stack", string(debug.Stack())),
					zap.String("path", c.Path()),
				)

				err = c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
					Error: "internal server error",
					Code:  "PANIC",
				})
			}
		}()

		return c.Next()
	}
}
