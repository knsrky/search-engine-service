package provider_b

import (
	"encoding/xml"
	"time"

	"search-engine-service/internal/domain"
)

// Feed represents the XML response from Provider B.
type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Items   Items    `xml:"items"`
	Meta    Meta     `xml:"meta"`
}

// Items wraps the list of items.
type Items struct {
	Items []Item `xml:"item"`
}

// Item represents a single content item from Provider B.
type Item struct {
	ID              string     `xml:"id"`
	Headline        string     `xml:"headline"`
	Type            string     `xml:"type"`
	Stats           Stats      `xml:"stats"`
	PublicationDate string     `xml:"publication_date"`
	Categories      Categories `xml:"categories"`
}

// Stats holds content metrics (varies by type).
type Stats struct {
	// Video stats
	Views    int    `xml:"views"`
	Likes    int    `xml:"likes"`
	Duration string `xml:"duration"`

	// Article stats
	ReadingTime int `xml:"reading_time"`
	Reactions   int `xml:"reactions"`
	Comments    int `xml:"comments"`
}

// Categories wraps the list of categories.
type Categories struct {
	Category []string `xml:"category"`
}

// Meta holds pagination info.
type Meta struct {
	TotalCount   int `xml:"total_count"`
	CurrentPage  int `xml:"current_page"`
	ItemsPerPage int `xml:"items_per_page"`
}

// ToDomain converts Item to domain.Content.
func (i *Item) ToDomain(providerID string) *domain.Content {
	// Parse date (format: 2024-03-15)
	publishedAt, _ := time.Parse("2006-01-02", i.PublicationDate)

	content := &domain.Content{
		ProviderID:  providerID,
		ExternalID:  i.ID,
		Title:       i.Headline,
		Type:        domain.ContentType(i.Type),
		Tags:        i.Categories.Category,
		PublishedAt: publishedAt,
	}

	// Set type-specific metrics
	if i.Type == "video" {
		content.Views = i.Stats.Views
		content.Likes = i.Stats.Likes
		content.Duration = i.Stats.Duration
	} else if i.Type == "article" {
		content.ReadingTime = i.Stats.ReadingTime
		content.Reactions = i.Stats.Reactions
		content.Comments = i.Stats.Comments
	}

	return content
}
