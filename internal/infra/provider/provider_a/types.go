package provider_a

import (
	"time"

	"search-engine-service/internal/domain"
)

// Response represents the JSON response from Provider A.
type Response struct {
	Contents   []ContentItem `json:"contents"`
	Pagination Pagination    `json:"pagination"`
}

// ContentItem represents a single content item from Provider A.
type ContentItem struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Metrics     Metrics  `json:"metrics"`
	PublishedAt string   `json:"published_at"`
	Tags        []string `json:"tags"`
}

// Metrics holds video metrics.
type Metrics struct {
	Views    int    `json:"views"`
	Likes    int    `json:"likes"`
	Duration string `json:"duration"`
}

// Pagination holds pagination info.
type Pagination struct {
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// ToDomain converts ContentItem to domain.Content.
func (c *ContentItem) ToDomain(providerID string) *domain.Content {
	publishedAt, _ := time.Parse(time.RFC3339, c.PublishedAt)

	return &domain.Content{
		ProviderID:  providerID,
		ExternalID:  c.ID,
		Title:       c.Title,
		Type:        domain.ContentType(c.Type),
		Tags:        c.Tags,
		Views:       c.Metrics.Views,
		Likes:       c.Metrics.Likes,
		Duration:    c.Metrics.Duration,
		PublishedAt: publishedAt,
	}
}
