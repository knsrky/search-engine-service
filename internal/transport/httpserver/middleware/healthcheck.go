// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"gorm.io/gorm"
)

// NewHealthCheck creates a Fiber healthcheck middleware with Kubernetes-style endpoints.
//
// Endpoints:
//   - GET /livez  - Liveness probe (app is running)
//   - GET /readyz - Readiness probe (app is ready to serve, DB connected)
//
// This middleware should be registered BEFORE other routes.
func NewHealthCheck(db *gorm.DB) fiber.Handler {
	return healthcheck.New(healthcheck.Config{
		// Liveness probe - is the application running?
		LivenessEndpoint: "/livez",
		LivenessProbe: func(_ *fiber.Ctx) bool {
			return true // Always return true if the app is running
		},

		// Readiness probe - is the application ready to serve traffic?
		ReadinessEndpoint: "/readyz",
		ReadinessProbe: func(_ *fiber.Ctx) bool {
			if db == nil {
				return false
			}
			sqlDB, err := db.DB()
			if err != nil {
				return false
			}

			return sqlDB.Ping() == nil
		},
	})
}
