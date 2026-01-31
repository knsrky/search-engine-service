// Package domain contains the core business logic and entities.
package domain

// ContentTypeCoefficient returns the scoring coefficient for content type.
// Video content is weighted higher than articles.
func ContentTypeCoefficient(contentType ContentType) float64 {
	switch contentType {
	case ContentTypeVideo:
		return 1.5
	case ContentTypeArticle:
		return 1.0
	default:
		return 1.0
	}
}

// CalculateScore computes the final relevance/popularity score for content.
//
// Formula:
//
//	Final Score = (Base Score * Content Type Coefficient) + Recency Score + Engagement Score
//
// Base Score:
//   - Video: views/1000 + likes/100
//   - Article: reading_time + reactions/50
//
// Content Type Coefficient:
//   - Video: 1.5
//   - Article: 1.0
//
// Recency Score:
//   - Within 1 week: +5
//   - Within 1 month: +3
//   - Within 3 months: +1
//   - Older: +0
//
// Engagement Score:
//   - Video: (likes/views) * 10
//   - Article: (reactions/reading_time) * 5
func CalculateScore(c *Content) float64 {
	if c == nil {
		return 0
	}

	baseScore := calculateBaseScore(c)
	typeCoeff := ContentTypeCoefficient(c.Type)
	recencyScore := calculateRecencyScore(c)
	engagementScore := calculateEngagementScore(c)

	finalScore := (baseScore * typeCoeff) + recencyScore + engagementScore

	// Round to 2 decimal places
	return roundTo2Decimals(finalScore)
}

// calculateBaseScore computes the base score based on content type.
//
// Video: views/1000 + likes/100
// Article: reading_time + reactions/50
func calculateBaseScore(c *Content) float64 {
	switch c.Type {
	case ContentTypeVideo:
		return float64(c.Views)/1000 + float64(c.Likes)/100
	case ContentTypeArticle:
		return float64(c.ReadingTime) + float64(c.Reactions)/50
	default:
		return 0
	}
}

// calculateRecencyScore returns a bonus based on content age.
//
//	1 week (7 days): +5
//	1 month (30 days): +3
//	3 months (90 days): +1
//	Older: +0
func calculateRecencyScore(c *Content) float64 {
	days := c.DaysSincePublished()

	switch {
	case days <= 7:
		return 5
	case days <= 30:
		return 3
	case days <= 90:
		return 1
	default:
		return 0
	}
}

// calculateEngagementScore computes engagement bonus based on content type.
//
// Video: (likes/views) * 10
// Article: (reactions/reading_time) * 5
func calculateEngagementScore(c *Content) float64 {
	switch c.Type {
	case ContentTypeVideo:
		if c.Views == 0 {
			return 0
		}
		return (float64(c.Likes) / float64(c.Views)) * 10
	case ContentTypeArticle:
		if c.ReadingTime == 0 {
			return 0
		}
		return (float64(c.Reactions) / float64(c.ReadingTime)) * 5
	default:
		return 0
	}
}

// roundTo2Decimals rounds a float to 2 decimal places.
func roundTo2Decimals(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
