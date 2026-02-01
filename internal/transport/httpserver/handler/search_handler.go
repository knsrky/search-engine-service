// Package handler provides HTTP handlers for the API.
package handler

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"search-engine-service/internal/app/service"
	"search-engine-service/internal/transport/httpserver/dto"
	"search-engine-service/internal/validator"
)

// SearchHandler handles search-related HTTP requests.
type SearchHandler struct {
	service   *service.SearchService
	validator *validator.Validator
	logger    *zap.Logger
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(svc *service.SearchService, v *validator.Validator, logger *zap.Logger) *SearchHandler {
	return &SearchHandler{
		service:   svc,
		validator: v,
		logger:    logger,
	}
}

// Search handles GET /api/v1/contents
func (h *SearchHandler) Search(c *fiber.Ctx) error {
	var req dto.SearchRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "invalid query parameters",
			Code:  "INVALID_PARAMS",
		})
	}

	if err := h.validator.Validate(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "validation failed",
			Code:    "VALIDATION_ERROR",
			Details: err,
		})
	}

	params := req.ToSearchParams()
	result, err := h.service.Search(c.Context(), params)
	if err != nil {
		h.logger.Error("search failed", zap.Error(err))

		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error: "search failed",
			Code:  "INTERNAL_ERROR",
		})
	}

	return c.JSON(dto.FromSearchResult(result))
}

// GetByID handles GET /api/v1/contents/:id
func (h *SearchHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "id is required",
			Code:  "MISSING_ID",
		})
	}

	content, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		h.logger.Error("get by id failed", zap.String("id", id), zap.Error(err))

		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error: "failed to get content",
			Code:  "INTERNAL_ERROR",
		})
	}

	if content == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
			Error: "content not found",
			Code:  "NOT_FOUND",
		})
	}

	return c.JSON(dto.FromDomainContent(content))
}
