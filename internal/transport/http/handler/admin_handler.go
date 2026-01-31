package handler

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"search-engine-service/internal/app/service"
	"search-engine-service/internal/transport/http/dto"
	"search-engine-service/internal/validator"
)

// AdminHandler handles admin-related HTTP requests.
type AdminHandler struct {
	syncService *service.SyncService
	validator   *validator.Validator
	logger      *zap.Logger
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(syncSvc *service.SyncService, v *validator.Validator, logger *zap.Logger) *AdminHandler {
	return &AdminHandler{
		syncService: syncSvc,
		validator:   v,
		logger:      logger,
	}
}

// SyncAll handles POST /api/v1/admin/sync
func (h *AdminHandler) SyncAll(c *fiber.Ctx) error {
	h.logger.Info("manual sync triggered")

	results := h.syncService.SyncAll(c.Context())

	return c.JSON(dto.FromSyncResults(results))
}

// SyncProvider handles POST /api/v1/admin/sync/:provider
func (h *AdminHandler) SyncProvider(c *fiber.Ctx) error {
	providerName := c.Params("provider")
	if providerName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "provider name is required",
			Code:  "MISSING_PROVIDER",
		})
	}

	h.logger.Info("manual provider sync triggered", zap.String("provider", providerName))

	result, err := h.syncService.SyncProvider(c.Context(), providerName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error: err.Error(),
			Code:  "SYNC_FAILED",
		})
	}

	if result == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
			Error: "provider not found",
			Code:  "PROVIDER_NOT_FOUND",
		})
	}

	return c.JSON(dto.SyncResultResponse{
		Provider: result.Provider,
		Count:    result.Count,
		Duration: result.Duration.String(),
	})
}

// GetProviders handles GET /api/v1/admin/providers
func (h *AdminHandler) GetProviders(c *fiber.Ctx) error {
	providers := h.syncService.GetProviderNames()
	return c.JSON(fiber.Map{
		"providers": providers,
	})
}
