package handler

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"search-engine-service/internal/app/service"
)

// DashboardHandler handles dashboard-related HTTP requests.
type DashboardHandler struct {
	searchService *service.SearchService
	logger        *zap.Logger
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(svc *service.SearchService, logger *zap.Logger) *DashboardHandler {
	return &DashboardHandler{
		searchService: svc,
		logger:        logger,
	}
}

// Render handles GET /dashboard
// Renders the dashboard HTML page using Fiber's template engine.
func (h *DashboardHandler) Render(c *fiber.Ctx) error {
	// Get content count for stats
	count, _ := h.searchService.Count(c.Context())

	return c.Render("pages/dashboard", fiber.Map{
		"Title":        "Search Engine Dashboard",
		"ContentCount": count,
	}, "layouts/base")
}
