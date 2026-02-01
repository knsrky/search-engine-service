// Package service provides application use cases.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"search-engine-service/internal/domain"
)

// SearchService handles content search operations.
type SearchService struct {
	repo     domain.ContentRepository
	cache    domain.Cache  // Optional cache (can be nil)
	cacheTTL time.Duration // TTL for cached search results
	logger   *zap.Logger
}

// NewSearchService creates a new SearchService.
// cache is optional and can be nil to disable caching.
// cacheTTL is only used if cache is not nil.
func NewSearchService(
	repo domain.ContentRepository,
	cache domain.Cache,
	cacheTTL time.Duration,
	logger *zap.Logger,
) *SearchService {
	return &SearchService{
		repo:     repo,
		cache:    cache,
		cacheTTL: cacheTTL,
		logger:   logger,
	}
}

// Search searches for contents based on the given parameters.
// Implements cache-aside pattern with TTL-based expiration.
func (s *SearchService) Search(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
	params.Validate()

	s.logger.Debug("searching contents",
		zap.String("query", params.Query),
		zap.String("type", string(params.Type)),
		zap.Int("page", params.Page),
		zap.Int("page_size", params.PageSize),
	)

	// Try cache if available
	if s.cache != nil {
		cacheKey := buildSearchCacheKey(params)
		if data, err := s.cache.Get(ctx, cacheKey); err == nil && data != nil {
			var result domain.SearchResult
			if err := json.Unmarshal(data, &result); err == nil {
				s.logger.Debug("cache hit",
					zap.String("key", cacheKey),
					zap.String("query", params.Query),
				)

				return &result, nil
			}
			// Unmarshal failed - continue to DB query
			s.logger.Warn("cache unmarshal failed",
				zap.String("key", cacheKey),
				zap.Error(err),
			)
		}
	}

	// Query database on cache miss or cache disabled
	result, err := s.repo.Search(ctx, params)
	if err != nil {
		s.logger.Error("search failed", zap.Error(err))

		return nil, err
	}

	s.logger.Debug("search completed",
		zap.Int64("total", result.Total),
		zap.Int("count", len(result.Contents)),
	)

	// Store in cache with TTL if cache is available
	if s.cache != nil {
		cacheKey := buildSearchCacheKey(params)
		if data, err := json.Marshal(result); err == nil {
			if err := s.cache.Set(ctx, cacheKey, data, s.cacheTTL); err != nil {
				// Don't fail the request on cache errors - log and continue
				s.logger.Warn("failed to cache search result",
					zap.Error(err),
					zap.String("key", cacheKey),
				)
			} else {
				s.logger.Debug("cached search result",
					zap.String("key", cacheKey),
					zap.Duration("ttl", s.cacheTTL),
				)
			}
		} else {
			s.logger.Warn("failed to marshal search result for caching",
				zap.Error(err),
				zap.String("key", cacheKey),
			)
		}
	}

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

// buildSearchCacheKey creates a deterministic cache key from search parameters.
// Format: search:query:type:page:pagesize:sortby:sortorder
func buildSearchCacheKey(params domain.SearchParams) string {
	return fmt.Sprintf("search:%s:%s:%d:%d:%s:%s",
		params.Query,
		params.Type,
		params.Page,
		params.PageSize,
		params.SortBy,
		params.SortOrder,
	)
}
