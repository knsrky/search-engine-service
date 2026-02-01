package domain

import (
	"math"
	"testing"
	"time"
)

const floatTolerance = 1e-9

func TestCalculateScore_Video(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		content  *Content
		expected float64
	}{
		{
			name: "popular recent video",
			content: &Content{
				Type:        ContentTypeVideo,
				Views:       100000, // 100000/1000 = 100
				Likes:       10000,  // 10000/100 = 100
				PublishedAt: now,    // +5 recency
				// Base: 100 + 100 = 200
				// TypeCoeff: 1.5 → 200 * 1.5 = 300
				// Engagement: (10000/100000) * 10 = 1.0
				// Final: 300 + 5 + 1 = 306
			},
			expected: 306.0,
		},
		{
			name: "low engagement video",
			content: &Content{
				Type:        ContentTypeVideo,
				Views:       10000, // 10000/1000 = 10
				Likes:       100,   // 100/100 = 1
				PublishedAt: now,   // +5 recency
				// Base: 10 + 1 = 11
				// TypeCoeff: 1.5 → 11 * 1.5 = 16.5
				// Engagement: (100/10000) * 10 = 0.1
				// Final: 16.5 + 5 + 0.1 = 21.6
			},
			expected: 21.6,
		},
		{
			name: "month old video",
			content: &Content{
				Type:        ContentTypeVideo,
				Views:       50000,                  // 50000/1000 = 50
				Likes:       5000,                   // 5000/100 = 50
				PublishedAt: now.AddDate(0, 0, -15), // +3 recency (within month)
				// Base: 50 + 50 = 100
				// TypeCoeff: 1.5 → 100 * 1.5 = 150
				// Engagement: (5000/50000) * 10 = 1.0
				// Final: 150 + 3 + 1 = 154
			},
			expected: 154.0,
		},
		{
			name: "old video (>90 days)",
			content: &Content{
				Type:        ContentTypeVideo,
				Views:       50000,
				Likes:       5000,
				PublishedAt: now.AddDate(0, 0, -100), // +0 recency
			},
			expected: 151.0, // 150 + 0 + 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateScore(tt.content)
			if score != tt.expected {
				t.Errorf("CalculateScore() = %v, want %v", score, tt.expected)
			}
		})
	}
}

func TestCalculateScore_Article(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		content  *Content
		expected float64
	}{
		{
			name: "popular recent article",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 8,   // 8
				Reactions:   500, // 500/50 = 10
				PublishedAt: now, // +5 recency
				// Base: 8 + 10 = 18
				// TypeCoeff: 1.0 → 18 * 1.0 = 18
				// Engagement: (500/8) * 5 = 312.5
				// Final: 18 + 5 + 312.5 = 335.5
			},
			expected: 335.5,
		},
		{
			name: "low reactions article",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 5,
				Reactions:   50,  // 50/50 = 1
				PublishedAt: now, // +5 recency
				// Base: 5 + 1 = 6
				// TypeCoeff: 1.0 → 6
				// Engagement: (50/5) * 5 = 50
				// Final: 6 + 5 + 50 = 61
			},
			expected: 61.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateScore(tt.content)
			if score != tt.expected {
				t.Errorf("CalculateScore() = %v, want %v", score, tt.expected)
			}
		})
	}
}

func TestCalculateScore_NilContent(t *testing.T) {
	score := CalculateScore(nil)
	if score != 0 {
		t.Errorf("CalculateScore(nil) = %v, want 0", score)
	}
}

func TestContentTypeCoefficient(t *testing.T) {
	tests := []struct {
		contentType ContentType
		expected    float64
	}{
		{ContentTypeVideo, 1.5},
		{ContentTypeArticle, 1.0},
		{"unknown", 1.0},
	}

	for _, tt := range tests {
		t.Run(string(tt.contentType), func(t *testing.T) {
			coeff := ContentTypeCoefficient(tt.contentType)
			if coeff != tt.expected {
				t.Errorf("ContentTypeCoefficient(%v) = %v, want %v", tt.contentType, coeff, tt.expected)
			}
		})
	}
}

func TestCalculateRecencyScore(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		publishedAt time.Time
		expected    float64
	}{
		{"today", now, 5},
		{"5 days ago", now.AddDate(0, 0, -5), 5},
		{"7 days ago", now.AddDate(0, 0, -7), 5},
		{"8 days ago", now.AddDate(0, 0, -8), 3},
		{"20 days ago", now.AddDate(0, 0, -20), 3},
		{"30 days ago", now.AddDate(0, 0, -30), 3},
		{"31 days ago", now.AddDate(0, 0, -31), 1},
		{"60 days ago", now.AddDate(0, 0, -60), 1},
		{"90 days ago", now.AddDate(0, 0, -90), 1},
		{"91 days ago", now.AddDate(0, 0, -91), 0},
		{"1 year ago", now.AddDate(-1, 0, 0), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := &Content{PublishedAt: tt.publishedAt}
			score := calculateRecencyScore(content)
			if score != tt.expected {
				t.Errorf("calculateRecencyScore() = %v, want %v", score, tt.expected)
			}
		})
	}
}

func TestCalculateEngagementScore_Video(t *testing.T) {
	tests := []struct {
		name     string
		views    int
		likes    int
		expected float64
	}{
		{"10% engagement", 10000, 1000, 1.0}, // (1000/10000) * 10 = 1.0
		{"5% engagement", 10000, 500, 0.5},   // (500/10000) * 10 = 0.5
		{"zero views", 0, 100, 0},            // Division by zero protection
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := &Content{
				Type:  ContentTypeVideo,
				Views: tt.views,
				Likes: tt.likes,
			}
			score := calculateEngagementScore(content)
			if score != tt.expected {
				t.Errorf("calculateEngagementScore() = %v, want %v", score, tt.expected)
			}
		})
	}
}

func TestCalculateEngagementScore_Article(t *testing.T) {
	tests := []struct {
		name        string
		readingTime int
		reactions   int
		expected    float64
	}{
		{"high engagement", 5, 100, 100.0}, // (100/5) * 5 = 100
		{"low engagement", 10, 20, 10.0},   // (20/10) * 5 = 10
		{"zero reading time", 0, 100, 0},   // Division by zero protection
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := &Content{
				Type:        ContentTypeArticle,
				ReadingTime: tt.readingTime,
				Reactions:   tt.reactions,
			}
			score := calculateEngagementScore(content)
			if score != tt.expected {
				t.Errorf("calculateEngagementScore() = %v, want %v", score, tt.expected)
			}
		})
	}
}

// Gap #1: Base Score Edge Cases
func TestCalculateScore_BaseScoreEdgeCases(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		content  *Content
		expected float64
	}{
		{
			name: "video with zero likes",
			content: &Content{
				Type:        ContentTypeVideo,
				Views:       10000, // 10000/1000 = 10
				Likes:       0,     // 0/100 = 0
				PublishedAt: now,
				// Base: 10 + 0 = 10
				// TypeCoeff: 1.5 → 10 * 1.5 = 15
				// Engagement: 0/10000 * 10 = 0
				// Recency: +5
				// Final: 15 + 5 + 0 = 20
			},
			expected: 20.0,
		},
		{
			name: "article with zero reactions",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 10,
				Reactions:   0, // 0/50 = 0
				PublishedAt: now,
				// Base: 10 + 0 = 10
				// TypeCoeff: 1.0 → 10
				// Engagement: 0/10 * 5 = 0
				// Recency: +5
				// Final: 10 + 5 + 0 = 15
			},
			expected: 15.0,
		},
		{
			name: "article with zero reading time (edge case)",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 0,
				Reactions:   100, // 100/50 = 2
				PublishedAt: now,
				// Base: 0 + 2 = 2
				// TypeCoeff: 1.0 → 2
				// Engagement: 0 (division by zero protection)
				// Recency: +5
				// Final: 2 + 5 + 0 = 7
			},
			expected: 7.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateScore(tt.content)
			if score != tt.expected {
				t.Errorf("CalculateScore() = %v, want %v", score, tt.expected)
			}
		})
	}
}

// Gap #2: Engagement Edge Cases
func TestCalculateEngagementScore_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  *Content
		expected float64
	}{
		{
			name: "video with 100% engagement (likes = views)",
			content: &Content{
				Type:  ContentTypeVideo,
				Views: 1000,
				Likes: 1000,
			},
			expected: 10.0, // (1000/1000) * 10 = 10
		},
		{
			name: "video with >100% engagement (likes > views)",
			content: &Content{
				Type:  ContentTypeVideo,
				Views: 1000,
				Likes: 1500,
			},
			expected: 15.0, // (1500/1000) * 10 = 15
		},
		{
			name: "video with very high engagement ratio",
			content: &Content{
				Type:  ContentTypeVideo,
				Views: 10,
				Likes: 100,
			},
			expected: 100.0, // (100/10) * 10 = 100
		},
		{
			name: "video with very low engagement ratio",
			content: &Content{
				Type:  ContentTypeVideo,
				Views: 1000000,
				Likes: 1,
			},
			expected: 0.00001, // (1/1000000) * 10 = 0.00001 (no rounding in engagement)
		},
		{
			name: "video with zero likes",
			content: &Content{
				Type:  ContentTypeVideo,
				Views: 10000,
				Likes: 0,
			},
			expected: 0.0,
		},
		{
			name: "article with 100% engagement (reactions = reading_time)",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 10,
				Reactions:   10,
			},
			expected: 5.0, // (10/10) * 5 = 5
		},
		{
			name: "article with >100% engagement (reactions > reading_time)",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 5,
				Reactions:   100,
			},
			expected: 100.0, // (100/5) * 5 = 100
		},
		{
			name: "article with very high engagement ratio",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 1,
				Reactions:   1000,
			},
			expected: 5000.0, // (1000/1) * 5 = 5000
		},
		{
			name: "article with very low engagement ratio",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 60,
				Reactions:   1,
			},
			expected: 0.08333333333333333, // (1/60) * 5 = 0.0833... (no rounding in engagement)
		},
		{
			name: "article with zero reactions",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 10,
				Reactions:   0,
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateEngagementScore(tt.content)
			if math.Abs(score-tt.expected) > floatTolerance {
				t.Errorf("calculateEngagementScore() = %v, want %v", score, tt.expected)
			}
		})
	}
}

// Gap #3: Extreme Values & Precision
func TestCalculateScore_ExtremeValuesAndPrecision(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		content  *Content
		expected float64
	}{
		{
			name: "video with 1M+ views",
			content: &Content{
				Type:        ContentTypeVideo,
				Views:       1000000, // 1000000/1000 = 1000
				Likes:       50000,   // 50000/100 = 500
				PublishedAt: now,
				// Base: 1000 + 500 = 1500
				// TypeCoeff: 1.5 → 1500 * 1.5 = 2250
				// Engagement: (50000/1000000) * 10 = 0.5
				// Recency: +5
				// Final: 2250 + 5 + 0.5 = 2255.5
			},
			expected: 2255.5,
		},
		{
			name: "article with extreme reactions",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 15,
				Reactions:   100000, // 100000/50 = 2000
				PublishedAt: now,
				// Base: 15 + 2000 = 2015
				// TypeCoeff: 1.0 → 2015
				// Engagement: (100000/15) * 5 = 33333.33...
				// Recency: +5
				// Final: 2015 + 5 + 33333.33 = 35353.33
			},
			expected: 35353.33,
		},
		{
			name: "rounding precision test - boundary value",
			content: &Content{
				Type:        ContentTypeVideo,
				Views:       3333, // 3333/1000 = 3.333
				Likes:       333,  // 333/100 = 3.33
				PublishedAt: now,
				// Base: 3.333 + 3.33 = 6.663
				// TypeCoeff: 1.5 → 6.663 * 1.5 = 9.9945
				// Engagement: (333/3333) * 10 = 0.999...
				// Recency: +5
				// Final: 9.9945 + 5 + 0.999 = 15.9935 → rounds to 15.99
			},
			expected: 15.99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateScore(tt.content)
			if score != tt.expected {
				t.Errorf("CalculateScore() = %v, want %v", score, tt.expected)
			}
		})
	}
}

// Gap #4: Type Safety & Data Validation
func TestCalculateScore_TypeMismatch(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		content  *Content
		expected float64
	}{
		{
			name: "video type with reading time set (ignored by type-based calculation)",
			content: &Content{
				Type:        ContentTypeVideo,
				Views:       10000, // 10000/1000 = 10
				Likes:       1000,  // 1000/100 = 10
				ReadingTime: 10,    // Ignored by calculateBaseScore switch
				Reactions:   100,   // Ignored by calculateEngagementScore switch
				PublishedAt: now,
				// Base: 10 + 10 = 20 (only views/likes used)
				// TypeCoeff: 1.5 → 20 * 1.5 = 30
				// Engagement: (1000/10000) * 10 = 1 (only likes/views used)
				// Recency: +5
				// Final: 30 + 5 + 1 = 36
			},
			expected: 36.0,
		},
		{
			name: "article type with views/likes set (should ignore)",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 10,
				Reactions:   100,
				Views:       10000, // Should be ignored for article
				Likes:       1000,  // Should be ignored for article
				PublishedAt: now,
			},
			expected: 67.0, // Article calculation: (10+2)*1.0 + 5 + 50 = 67
		},
		{
			name: "unknown content type with metrics",
			content: &Content{
				Type:        "unknown",
				Views:       10000,
				Likes:       1000,
				ReadingTime: 10,
				Reactions:   100,
				PublishedAt: now,
			},
			expected: 5.0, // Unknown type: base=0, coeff=1.0, recency=5, engagement=0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateScore(tt.content)
			if score != tt.expected {
				t.Errorf("CalculateScore() = %v, want %v", score, tt.expected)
			}
		})
	}
}

// Gap #5: Integration Edge Cases
func TestCalculateScore_IntegrationEdgeCases(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		content  *Content
		expected float64
	}{
		{
			name: "old content + zero engagement + zero base (video)",
			content: &Content{
				Type:        ContentTypeVideo,
				Views:       0,
				Likes:       0,
				PublishedAt: now.AddDate(0, 0, -100),
				// Base: 0 + 0 = 0
				// TypeCoeff: 1.5 → 0
				// Engagement: 0
				// Recency: 0 (>90 days)
				// Final: 0 + 0 + 0 = 0
			},
			expected: 0.0,
		},
		{
			name: "old content + zero engagement + zero base (article)",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 0,
				Reactions:   0,
				PublishedAt: now.AddDate(0, 0, -100),
				// Base: 0 + 0 = 0
				// TypeCoeff: 1.0 → 0
				// Engagement: 0
				// Recency: 0 (>90 days)
				// Final: 0 + 0 + 0 = 0
			},
			expected: 0.0,
		},
		{
			name: "boundary test - exactly 7 days + high engagement",
			content: &Content{
				Type:        ContentTypeVideo,
				Views:       1000,
				Likes:       1000, // 100% engagement
				PublishedAt: now.AddDate(0, 0, -7),
				// Base: (1000/1000 + 1000/100) = 1 + 10 = 11
				// TypeCoeff: 1.5 → 11 * 1.5 = 16.5
				// Engagement: (1000/1000) * 10 = 10
				// Recency: 5 (exactly 7 days)
				// Final: 16.5 + 5 + 10 = 31.5
			},
			expected: 31.5,
		},
		{
			name: "boundary test - exactly 30 days + medium engagement",
			content: &Content{
				Type:        ContentTypeArticle,
				ReadingTime: 10,
				Reactions:   50,
				PublishedAt: now.AddDate(0, 0, -30),
				// Base: 10 + 50/50 = 10 + 1 = 11
				// TypeCoeff: 1.0 → 11
				// Engagement: (50/10) * 5 = 25
				// Recency: 3 (exactly 30 days)
				// Final: 11 + 3 + 25 = 39
			},
			expected: 39.0,
		},
		{
			name: "boundary test - exactly 90 days + low engagement",
			content: &Content{
				Type:        ContentTypeVideo,
				Views:       10000,
				Likes:       100,
				PublishedAt: now.AddDate(0, 0, -90),
				// Base: (10000/1000 + 100/100) = 10 + 1 = 11
				// TypeCoeff: 1.5 → 11 * 1.5 = 16.5
				// Engagement: (100/10000) * 10 = 0.1
				// Recency: 1 (exactly 90 days)
				// Final: 16.5 + 1 + 0.1 = 17.6
			},
			expected: 17.6,
		},
		{
			name: "coefficient application order verification",
			content: &Content{
				Type:        ContentTypeVideo,
				Views:       2000, // 2000/1000 = 2
				Likes:       200,  // 200/100 = 2
				PublishedAt: now,
				// Base: 2 + 2 = 4
				// TypeCoeff: 1.5 → 4 * 1.5 = 6 (coefficient applied BEFORE additions)
				// Engagement: (200/2000) * 10 = 1
				// Recency: 5
				// Final: 6 + 5 + 1 = 12
				// If coefficient applied after: (4 + 5 + 1) * 1.5 = 15 (WRONG)
			},
			expected: 12.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateScore(tt.content)
			if score != tt.expected {
				t.Errorf("CalculateScore() = %v, want %v", score, tt.expected)
			}
		})
	}
}
