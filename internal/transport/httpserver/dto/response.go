package dto

import (
	"time"

	"search-engine-service/internal/app/service"
	"search-engine-service/internal/domain"
)

// ContentResponse represents a single content item in the response.
type ContentResponse struct {
	ID         string   `json:"id"`
	ProviderID string   `json:"provider_id"`
	ExternalID string   `json:"external_id"`
	Title      string   `json:"title"`
	Type       string   `json:"type"`
	Tags       []string `json:"tags,omitempty"`

	// Metrics
	Views       int    `json:"views,omitempty"`
	Likes       int    `json:"likes,omitempty"`
	Duration    string `json:"duration,omitempty"`
	ReadingTime int    `json:"reading_time,omitempty"`
	Reactions   int    `json:"reactions,omitempty"`
	Comments    int    `json:"comments,omitempty"`

	// Score
	Score float64 `json:"score"`

	// Timestamps
	PublishedAt string `json:"published_at"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// FromDomainContent converts domain.Content to ContentResponse.
func FromDomainContent(c *domain.Content) ContentResponse {
	return ContentResponse{
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
		PublishedAt: c.PublishedAt.Format(time.RFC3339),
		CreatedAt:   c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   c.UpdatedAt.Format(time.RFC3339),
	}
}

// SearchResponse represents the search results response.
type SearchResponse struct {
	Contents   []ContentResponse `json:"contents"`
	Pagination PaginationMeta    `json:"pagination"`
}

// PaginationMeta holds pagination metadata.
type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// FromSearchResult converts domain.SearchResult to SearchResponse.
func FromSearchResult(result *domain.SearchResult) SearchResponse {
	contents := make([]ContentResponse, len(result.Contents))
	for i, c := range result.Contents {
		contents[i] = FromDomainContent(c)
	}

	return SearchResponse{
		Contents: contents,
		Pagination: PaginationMeta{
			Total:      result.Total,
			Page:       result.Page,
			PageSize:   result.PageSize,
			TotalPages: result.TotalPages,
		},
	}
}

// SyncResultResponse represents the response for a sync operation.
type SyncResultResponse struct {
	Provider string `json:"provider"`
	Count    int    `json:"count"`
	Duration string `json:"duration"`
	Error    string `json:"error,omitempty"`
}

// SyncResponse represents the response for sync all operation.
type SyncResponse struct {
	Results []SyncResultResponse `json:"results"`
	Summary SyncSummary          `json:"summary"`
}

// SyncSummary holds summary of sync operation.
type SyncSummary struct {
	TotalSynced   int `json:"total_synced"`
	ProvidersOK   int `json:"providers_ok"`
	ProvidersFail int `json:"providers_fail"`
}

// FromSyncResults converts service.SyncResult slice to SyncResponse.
func FromSyncResults(results []service.SyncResult) SyncResponse {
	resp := SyncResponse{
		Results: make([]SyncResultResponse, len(results)),
	}

	for i, r := range results {
		errMsg := ""
		if r.Error != nil {
			errMsg = r.Error.Error()
			resp.Summary.ProvidersFail++
		} else {
			resp.Summary.TotalSynced += r.Count
			resp.Summary.ProvidersOK++
		}

		resp.Results[i] = SyncResultResponse{
			Provider: r.Provider,
			Count:    r.Count,
			Duration: r.Duration.String(),
			Error:    errMsg,
		}
	}

	return resp
}

// HealthResponse represents health check response.
type HealthResponse struct {
	Status    string            `json:"status"`
	Checks    map[string]string `json:"checks,omitempty"`
	Timestamp string            `json:"timestamp"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string      `json:"error"`
	Code    string      `json:"code,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// StatsResponse represents dashboard stats.
type StatsResponse struct {
	TotalContents int64            `json:"total_contents"`
	ByType        map[string]int64 `json:"by_type"`
	ByProvider    map[string]int64 `json:"by_provider"`
}
