package domain

import (
	"context"
	"time"
)

// ContentRepository defines the interface for content persistence operations.
// Implementations: internal/infra/postgres/repository.go
type ContentRepository interface {
	// Search finds contents matching the given search parameters.
	Search(ctx context.Context, params SearchParams) (*SearchResult, error)

	// GetByID retrieves a single content by its internal ID.
	GetByID(ctx context.Context, id string) (*Content, error)

	// GetByProviderAndExternalID retrieves content by provider and external ID.
	GetByProviderAndExternalID(ctx context.Context, providerID, externalID string) (*Content, error)

	// Upsert creates or updates a single content.
	// Uses provider_id + external_id as the unique key.
	Upsert(ctx context.Context, content *Content) error

	// BulkUpsert creates or updates multiple contents in a batch.
	BulkUpsert(ctx context.Context, contents []*Content) error

	// Delete removes a content by its internal ID.
	Delete(ctx context.Context, id string) error

	// Count returns the total number of contents matching optional filters.
	Count(ctx context.Context, params SearchParams) (int64, error)
}

// Provider defines the interface for external content providers.
// Implementations: internal/infra/provider/provider_a/, internal/infra/provider/provider_b/
type Provider interface {
	// Name returns the unique identifier for this provider.
	Name() string

	// Fetch retrieves all available content from the provider.
	// The implementation should handle pagination internally if needed.
	Fetch(ctx context.Context) ([]*Content, error)

	// HealthCheck verifies the provider is accessible.
	HealthCheck(ctx context.Context) error
}

// Cache defines the interface for caching operations.
// Implementations: internal/infra/cache/memory.go (optional)
type Cache interface {
	// Get retrieves a value by key. Returns nil if not found.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with the given TTL.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value by key.
	Delete(ctx context.Context, key string) error

	// Clear removes all cached values.
	Clear(ctx context.Context) error
}
