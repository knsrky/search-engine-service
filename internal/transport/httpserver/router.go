// Package httpserver provides HTTP server and routing.
package httpserver

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/template/html/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"search-engine-service/internal/app/service"
	"search-engine-service/internal/transport/httpserver/handler"
	"search-engine-service/internal/transport/httpserver/middleware"
	"search-engine-service/internal/validator"
)

// ServerConfig holds server configuration.
type ServerConfig struct {
	Port      int
	BodyLimit int
	Debug     bool
}

// Server wraps Fiber app with handlers.
type Server struct {
	App    *fiber.App
	Logger *zap.Logger
}

// NewServer creates a new HTTP server with all routes configured.
func NewServer(
	cfg ServerConfig,
	searchSvc *service.SearchService,
	syncSvc *service.SyncService,
	db *gorm.DB,
	v *validator.Validator,
	logger *zap.Logger,
) *Server {
	// Template engine for dashboard
	engine := html.New("./web/templates", ".html")
	if cfg.Debug {
		engine.Reload(true)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "search-engine-service",
		BodyLimit:    cfg.BodyLimit,
		ErrorHandler: errorHandler(logger),
		Views:        engine,
	})

	// Health check middleware MUST be registered BEFORE other middleware
	// for Kubernetes probes to work even during high load
	app.Use(middleware.NewHealthCheck(db))

	// Global middleware
	app.Use(requestid.New())
	app.Use(middleware.Recover(logger))
	app.Use(middleware.Logger(logger))
	app.Use(middleware.CORS())
	app.Use(compress.New())

	// Static files
	app.Static("/static", "./web/static")

	// Create handlers
	searchHandler := handler.NewSearchHandler(searchSvc, v, logger)
	adminHandler := handler.NewAdminHandler(syncSvc, v, logger)
	dashboardHandler := handler.NewDashboardHandler(searchSvc, logger)

	// Register routes
	registerRoutes(app, searchHandler, adminHandler, dashboardHandler)

	return &Server{
		App:    app,
		Logger: logger,
	}
}

// registerRoutes sets up all API routes.
func registerRoutes(
	app *fiber.App,
	searchHandler *handler.SearchHandler,
	adminHandler *handler.AdminHandler,
	dashboardHandler *handler.DashboardHandler,
) {
	// Health checks are handled by middleware (/livez, /readyz)

	// Dashboard (HTML)
	app.Get("/dashboard", dashboardHandler.Render)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/dashboard")
	})

	// API v1 routes
	v1 := app.Group("/api/v1")

	// Contents
	contents := v1.Group("/contents")
	contents.Get("/", searchHandler.Search)
	contents.Get("/:id", searchHandler.GetByID)

	// Admin routes
	admin := v1.Group("/admin")
	admin.Post("/sync", adminHandler.SyncAll)
	admin.Post("/sync/:provider", adminHandler.SyncProvider)
	admin.Get("/providers", adminHandler.GetProviders)
}

// errorHandler returns a custom error handler that logs based on HTTP status code.
// 404s are logged at DEBUG level (expected client behavior), 4xx at WARN, 5xx at ERROR.
func errorHandler(logger *zap.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		// Log based on status code - 404s are common and not server errors
		switch {
		case code == fiber.StatusNotFound:
			logger.Debug("resource not found",
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
			)
		case code >= 500:
			logger.Error("server error",
				zap.Error(err),
				zap.Int("status", code),
				zap.String("path", c.Path()),
			)
		case code >= 400:
			logger.Warn("client error",
				zap.Error(err),
				zap.Int("status", code),
				zap.String("path", c.Path()),
			)
		default:
			logger.Error("unhandled error",
				zap.Error(err),
				zap.Int("status", code),
				zap.String("path", c.Path()),
			)
		}

		return c.Status(code).JSON(fiber.Map{
			"error": err.Error(),
			"code":  "UNHANDLED_ERROR",
		})
	}
}

// Start starts the HTTP server.
func (s *Server) Start(port int) error {
	s.Logger.Info("starting HTTP server", zap.Int("port", port))

	return s.App.Listen(fmt.Sprintf(":%d", port))
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	s.Logger.Info("shutting down HTTP server")

	return s.App.Shutdown()
}
