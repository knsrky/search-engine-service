// Package dto provides Data Transfer Objects for HTTP requests and responses.
package dto

import "search-engine-service/internal/domain"

// SearchRequest represents the query parameters for searching contents.
type SearchRequest struct {
	Query     string `query:"q" validate:"max=200"`
	Type      string `query:"type" validate:"omitempty,oneof=video article"`
	SortBy    string `query:"sort_by" validate:"omitempty,oneof=relevance score published_at"`
	SortOrder string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
	Page      int    `query:"page" validate:"omitempty,min=1"`
	PageSize  int    `query:"page_size" validate:"omitempty,min=1,max=100"`
}

// ToSearchParams converts SearchRequest to domain.SearchParams.
// When a search query is provided and no explicit sort_by is specified,
// defaults to relevance sorting for optimal search experience.
func (r *SearchRequest) ToSearchParams() domain.SearchParams {
	params := domain.DefaultSearchParams()

	params.Query = r.Query
	params.Type = domain.ContentType(r.Type)

	if r.SortBy != "" {
		params.SortBy = domain.SortField(r.SortBy)
	} else if r.Query != "" {
		// Smart default: use relevance sort when searching
		params.SortBy = domain.SortFieldRelevance
	}

	if r.SortOrder != "" {
		params.SortOrder = domain.SortOrder(r.SortOrder)
	}
	if r.Page > 0 {
		params.Page = r.Page
	}
	if r.PageSize > 0 {
		params.PageSize = r.PageSize
	}

	return params
}

// SyncRequest represents the request body for manual sync.
type SyncRequest struct {
	Provider string `json:"provider" validate:"omitempty,max=50"`
}
