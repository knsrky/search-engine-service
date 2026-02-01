package domain

import (
	"testing"
	"time"
)

func TestNewContent(t *testing.T) {
	content := NewContent("provider_a", "v1", "Test Video", ContentTypeVideo)

	if content.ProviderID != "provider_a" {
		t.Errorf("expected provider_id 'provider_a', got %q", content.ProviderID)
	}
	if content.ExternalID != "v1" {
		t.Errorf("expected external_id 'v1', got %q", content.ExternalID)
	}
	if content.Title != "Test Video" {
		t.Errorf("expected title 'Test Video', got %q", content.Title)
	}
	if content.Type != ContentTypeVideo {
		t.Errorf("expected type 'video', got %q", content.Type)
	}
	if content.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestContent_IsVideo(t *testing.T) {
	video := &Content{Type: ContentTypeVideo}
	article := &Content{Type: ContentTypeArticle}

	if !video.IsVideo() {
		t.Error("expected IsVideo() to return true for video")
	}
	if video.IsArticle() {
		t.Error("expected IsArticle() to return false for video")
	}
	if article.IsVideo() {
		t.Error("expected IsVideo() to return false for article")
	}
	if !article.IsArticle() {
		t.Error("expected IsArticle() to return true for article")
	}
}

func TestContent_EngagementRate(t *testing.T) {
	tests := []struct {
		name     string
		content  *Content
		expected float64
	}{
		{
			name: "normal video",
			content: &Content{
				Type:  ContentTypeVideo,
				Views: 1000,
				Likes: 100,
			},
			expected: 0.1, // 100/1000
		},
		{
			name: "zero views",
			content: &Content{
				Type:  ContentTypeVideo,
				Views: 0,
				Likes: 100,
			},
			expected: 0,
		},
		{
			name: "article type",
			content: &Content{
				Type:  ContentTypeArticle,
				Views: 1000,
				Likes: 100,
			},
			expected: 0, // Not applicable to articles
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.content.EngagementRate()
			if got != tt.expected {
				t.Errorf("EngagementRate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestContent_DaysSincePublished(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		publishedAt time.Time
		minExpected int
		maxExpected int
	}{
		{
			name:        "published today",
			publishedAt: now,
			minExpected: 0,
			maxExpected: 0,
		},
		{
			name:        "published 7 days ago",
			publishedAt: now.AddDate(0, 0, -7),
			minExpected: 6, // Allow some margin
			maxExpected: 8,
		},
		{
			name:        "published in future",
			publishedAt: now.AddDate(0, 0, 1),
			minExpected: 0,
			maxExpected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := &Content{PublishedAt: tt.publishedAt}
			got := content.DaysSincePublished()
			if got < tt.minExpected || got > tt.maxExpected {
				t.Errorf("DaysSincePublished() = %d, want between %d and %d", got, tt.minExpected, tt.maxExpected)
			}
		})
	}
}
