package postgres

import (
	"time"

	"search-engine-service/internal/domain"

	"github.com/lib/pq"
)

// ContentModel is the GORM model for the contents table.
type ContentModel struct {
	ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProviderID string         `gorm:"type:varchar(50);not null;index:idx_provider_external,unique"`
	ExternalID string         `gorm:"type:varchar(100);not null;index:idx_provider_external,unique"`
	Title      string         `gorm:"type:varchar(500);not null"`
	Type       string         `gorm:"type:varchar(20);not null;index"`
	Tags       pq.StringArray `gorm:"type:text[]"`

	// Metrics
	Views       int    `gorm:"default:0"`
	Likes       int    `gorm:"default:0"`
	Duration    string `gorm:"type:varchar(20)"`
	ReadingTime int    `gorm:"default:0"`
	Reactions   int    `gorm:"default:0"`
	Comments    int    `gorm:"default:0"`

	// Score
	Score float64 `gorm:"type:decimal(10,2);default:0;index"`

	// Timestamps
	PublishedAt time.Time `gorm:"not null;index"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for ContentModel.
func (ContentModel) TableName() string {
	return "contents"
}

// ToDomain converts ContentModel to domain.Content.
func (m *ContentModel) ToDomain() *domain.Content {
	return &domain.Content{
		ID:          m.ID,
		ProviderID:  m.ProviderID,
		ExternalID:  m.ExternalID,
		Title:       m.Title,
		Type:        domain.ContentType(m.Type),
		Tags:        m.Tags,
		Views:       m.Views,
		Likes:       m.Likes,
		Duration:    m.Duration,
		ReadingTime: m.ReadingTime,
		Reactions:   m.Reactions,
		Comments:    m.Comments,
		Score:       m.Score,
		PublishedAt: m.PublishedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// FromDomain creates a ContentModel from domain.Content.
func FromDomain(c *domain.Content) *ContentModel {
	return &ContentModel{
		ID:          c.ID,
		ProviderID:  c.ProviderID,
		ExternalID:  c.ExternalID,
		Title:       c.Title,
		Type:        string(c.Type),
		Tags:        c.Tags,
		Views:       c.Views,
		Likes:       c.Likes,
		Duration:    c.Duration,
		ReadingTime: c.ReadingTime,
		Reactions:   c.Reactions,
		Comments:    c.Comments,
		Score:       c.Score,
		PublishedAt: c.PublishedAt,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// FromDomainSlice converts a slice of domain.Content to ContentModels.
func FromDomainSlice(contents []*domain.Content) []*ContentModel {
	models := make([]*ContentModel, len(contents))
	for i, c := range contents {
		models[i] = FromDomain(c)
	}

	return models
}
