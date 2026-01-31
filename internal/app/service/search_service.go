// Package service provides application use cases.
package service

import (
	"context"

	"go.uber.org/zap"

	"search-engine-service/internal/domain"
)

// SearchService handles content search operations.
type SearchService struct {
	repo   domain.ContentRepository
	logger *zap.Logger
}

// NewSearchService creates a new SearchService.
func NewSearchService(repo domain.ContentRepository, logger *zap.Logger) *SearchService {
	return &SearchService{
		repo:   repo,
		logger: logger,
	}
}

// Search searches for contents based on the given parameters.
func (s *SearchService) Search(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
	params.Validate()

	s.logger.Debug("searching contents",
		zap.String("query", params.Query),
		zap.String("type", string(params.Type)),
		zap.Int("page", params.Page),
		zap.Int("page_size", params.PageSize),
	)

	result, err := s.repo.Search(ctx, params)
	if err != nil {
		s.logger.Error("search failed", zap.Error(err))
		return nil, err
	}

	s.logger.Debug("search completed",
		zap.Int64("total", result.Total),
		zap.Int("count", len(result.Contents)),
	)

	return result, nil
}

// GetByID retrieves a single content by its internal ID.
func (s *SearchService) GetByID(ctx context.Context, id string) (*domain.Content, error) {
	content, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("get by id failed", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	return content, nil
}

// Count returns the total number of contents.
func (s *SearchService) Count(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx, domain.SearchParams{})
}
