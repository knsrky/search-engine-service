package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"search-engine-service/internal/domain"
)

// Repository implements domain.ContentRepository using PostgreSQL.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new PostgreSQL repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Search finds contents matching the given search parameters.
func (r *Repository) Search(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
	params.Validate()

	var models []ContentModel
	query := r.buildSearchQuery(params)

	// Get total count
	var total int64
	if err := query.WithContext(ctx).Model(&ContentModel{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("counting contents: %w", err)
	}

	// Build final query with pagination
	finalQuery := query.WithContext(ctx).
		Offset(params.Offset()).
		Limit(params.Limit())

	// Apply ordering (handles FTS relevance ranking safely)
	finalQuery = r.applyOrdering(finalQuery, params)

	// Execute query
	if err := finalQuery.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("searching contents: %w", err)
	}

	// Convert to domain
	contents := make([]*domain.Content, len(models))
	for i, m := range models {
		contents[i] = m.ToDomain()
	}

	return domain.NewSearchResult(contents, total, params), nil
}

// GetByID retrieves a single content by its internal ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Content, error) {
	var model ContentModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found
		}

		return nil, fmt.Errorf("getting content by id: %w", err)
	}

	return model.ToDomain(), nil
}

// GetByProviderAndExternalID retrieves content by provider and external ID.
func (r *Repository) GetByProviderAndExternalID(ctx context.Context, providerID, externalID string) (*domain.Content, error) {
	var model ContentModel
	err := r.db.WithContext(ctx).
		Where("provider_id = ? AND external_id = ?", providerID, externalID).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found
		}

		return nil, fmt.Errorf("getting content by provider and external id: %w", err)
	}

	return model.ToDomain(), nil
}

// Upsert creates or updates a single content.
func (r *Repository) Upsert(ctx context.Context, content *domain.Content) error {
	model := FromDomain(content)
	model.UpdatedAt = time.Now().UTC()

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider_id"}, {Name: "external_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "type", "tags",
			"views", "likes", "duration", "reading_time", "reactions", "comments",
			"score", "published_at", "updated_at",
		}),
	}).Create(model).Error

	if err != nil {
		return fmt.Errorf("upserting content: %w", err)
	}

	// Update the domain object with database-generated fields
	content.ID = model.ID
	content.CreatedAt = model.CreatedAt
	content.UpdatedAt = model.UpdatedAt

	return nil
}

// BulkUpsert creates or updates multiple contents in a batch.
func (r *Repository) BulkUpsert(ctx context.Context, contents []*domain.Content) error {
	if len(contents) == 0 {
		return nil
	}

	now := time.Now().UTC()
	models := FromDomainSlice(contents)
	for _, m := range models {
		m.UpdatedAt = now
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider_id"}, {Name: "external_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "type", "tags",
			"views", "likes", "duration", "reading_time", "reactions", "comments",
			"score", "published_at", "updated_at",
		}),
	}).CreateInBatches(models, 100).Error

	if err != nil {
		return fmt.Errorf("bulk upserting contents: %w", err)
	}

	// Update domain objects with database-generated fields
	for i, m := range models {
		contents[i].ID = m.ID
		contents[i].CreatedAt = m.CreatedAt
		contents[i].UpdatedAt = m.UpdatedAt
	}

	return nil
}

// Delete removes a content by its internal ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&ContentModel{})
	if result.Error != nil {
		return fmt.Errorf("deleting content: %w", result.Error)
	}

	return nil
}

// Count returns the total number of contents matching optional filters.
func (r *Repository) Count(ctx context.Context, params domain.SearchParams) (int64, error) {
	var count int64
	query := r.buildSearchQuery(params)
	if err := query.WithContext(ctx).Model(&ContentModel{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting contents: %w", err)
	}

	return count, nil
}

// buildSearchQuery builds the WHERE clause for search.
// When query is provided, uses PostgreSQL FTS with tsvector matching.
// All parameters are safely bound using GORM's parameterized queries.
func (r *Repository) buildSearchQuery(params domain.SearchParams) *gorm.DB {
	query := r.db.Model(&ContentModel{})

	// Full-Text Search: Use tsvector @@ tsquery when query provided
	// websearch_to_tsquery supports user-friendly syntax:
	// - "word1 word2" → word1 AND word2
	// - "word1 OR word2" → word1 OR word2
	// - "-word" → NOT word
	if params.Query != "" {
		query = query.Where(
			"search_vector @@ websearch_to_tsquery('english', ?)",
			params.Query,
		)
	}

	// Filter by content type
	if params.Type != "" {
		query = query.Where("type = ?", string(params.Type))
	}

	return query
}

// applyOrdering adds ORDER BY clause to the query.
//
// For relevance sort with a search query, uses hybrid ranking:
//
//	Rank = ts_rank × LOG(score + 10)
//
// This formula balances text relevance and popularity:
//
// | Scenario                   | ts_rank | Score     | Result              |
// |----------------------------|---------|-----------|---------------------|
// | Perfect match, new content | 0.9     | 0         | 0.9 × 1.0 = 0.9     |
// | Good match, popular        | 0.6     | 10,000    | 0.6 × 4.0 = 2.4     |
// | Poor match, viral          | 0.1     | 1,000,000 | 0.1 × 6.0 = 0.6     |
//
// Key insight: Perfect match of new content (0.9) beats poor match of viral (0.6)
func (r *Repository) applyOrdering(query *gorm.DB, params domain.SearchParams) *gorm.DB {
	direction := "DESC"
	if params.SortOrder == domain.SortOrderAsc {
		direction = "ASC"
	}

	switch params.SortBy {
	case domain.SortFieldRelevance:
		if params.Query != "" {
			// Use gorm.Expr with parameterized query for SQL injection safety.
			// This prevents injection from user input like "O'Reilly"
			// Uses cached log_score_cached column for efficient ranking
			expr := gorm.Expr(
				"(ts_rank(search_vector, websearch_to_tsquery('english', ?)) * log_score_cached) "+direction,
				params.Query,
			)

			return query.Clauses(clause.OrderBy{Expression: expr})
		}
		// Fallback to score when no query provided
		return query.Order("score " + direction)

	case domain.SortFieldScore:
		return query.Order("score " + direction)
	case domain.SortFieldPublishedAt:
		return query.Order("published_at " + direction)
	default:
		return query.Order("score " + direction)
	}
}
