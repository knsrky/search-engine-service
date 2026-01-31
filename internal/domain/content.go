// Package domain contains the core business logic and entities.
// This package has no external dependencies (only stdlib).
package domain

import (
	"time"
)

// ContentType represents the type of content.
type ContentType string

const (
	ContentTypeVideo   ContentType = "video"
	ContentTypeArticle ContentType = "article"
)

// Content represents a unified content entity from any provider.
// This is the core domain entity used throughout the application.
type Content struct {
	// Primary identifiers
	ID         string `json:"id"`          // Internal UUID
	ProviderID string `json:"provider_id"` // e.g., "provider_a", "provider_b"
	ExternalID string `json:"external_id"` // ID from the provider (unique per provider)

	// Content metadata
	Title string      `json:"title"`
	Type  ContentType `json:"type"` // video, article
	Tags  []string    `json:"tags,omitempty"`

	// Metrics (varies by content type)
	Views       int    `json:"views,omitempty"`        // Video: view count
	Likes       int    `json:"likes,omitempty"`        // Video: like count
	Duration    string `json:"duration,omitempty"`     // Video: duration (e.g., "15:30")
	ReadingTime int    `json:"reading_time,omitempty"` // Article: reading time in minutes
	Reactions   int    `json:"reactions,omitempty"`    // Article: reaction count
	Comments    int    `json:"comments,omitempty"`     // Article: comment count

	// Calculated scores
	Score float64 `json:"score"` // Calculated relevance/popularity score

	// Timestamps
	PublishedAt time.Time `json:"published_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewContent creates a new Content with generated ID and timestamps.
func NewContent(providerID, externalID, title string, contentType ContentType) *Content {
	now := time.Now().UTC()
	return &Content{
		ProviderID:  providerID,
		ExternalID:  externalID,
		Title:       title,
		Type:        contentType,
		Tags:        []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
		PublishedAt: now,
	}
}

// IsVideo returns true if content is a video.
func (c *Content) IsVideo() bool {
	return c.Type == ContentTypeVideo
}

// IsArticle returns true if content is an article.
func (c *Content) IsArticle() bool {
	return c.Type == ContentTypeArticle
}

// EngagementRate calculates the engagement rate for videos.
// Returns 0 for non-video content or if views is 0.
func (c *Content) EngagementRate() float64 {
	if !c.IsVideo() || c.Views == 0 {
		return 0
	}
	return float64(c.Likes) / float64(c.Views)
}

// DaysSincePublished returns the number of days since publication.
func (c *Content) DaysSincePublished() int {
	days := time.Since(c.PublishedAt).Hours() / 24
	if days < 0 {
		return 0
	}
	return int(days)
}
