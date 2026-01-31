package domain

// SortOrder represents the sort direction.
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// SortField represents the field to sort by.
type SortField string

const (
	SortFieldRelevance   SortField = "relevance" // FTS hybrid ranking: ts_rank × LOG(score + 10)
	SortFieldScore       SortField = "score"
	SortFieldPublishedAt SortField = "published_at"
)

// SearchParams holds search and filter parameters for content queries.
type SearchParams struct {
	// Text search
	Query string // Full-text search query

	// Filters
	Type ContentType // Filter by content type (video, article)

	// Sorting
	SortBy    SortField // Field to sort by (default: score)
	SortOrder SortOrder // Sort direction (default: desc)

	// Pagination
	Page     int // Page number (1-indexed)
	PageSize int // Items per page
}

// DefaultSearchParams returns search params with sensible defaults.
func DefaultSearchParams() SearchParams {
	return SearchParams{
		SortBy:    SortFieldScore,
		SortOrder: SortOrderDesc,
		Page:      1,
		PageSize:  5, // for limited dataset
	}
}

// Validate ensures search params are within acceptable bounds. This is bound correction, not validation.
func (p *SearchParams) Validate() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	if p.SortBy == "" {
		p.SortBy = SortFieldScore
	}
	if p.SortOrder == "" {
		p.SortOrder = SortOrderDesc
	}
}

// Offset calculates the database offset for pagination.
func (p *SearchParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the page size (alias for clarity).
func (p *SearchParams) Limit() int {
	return p.PageSize
}

// SearchResult holds paginated search results.
type SearchResult struct {
	Contents   []*Content `json:"contents"`
	Total      int64      `json:"total"`       // Total matching records
	Page       int        `json:"page"`        // Current page (1-indexed)
	PageSize   int        `json:"page_size"`   // Items per page
	TotalPages int        `json:"total_pages"` // Total number of pages
}

// NewSearchResult creates a new SearchResult with calculated pagination.
func NewSearchResult(contents []*Content, total int64, params SearchParams) *SearchResult {
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &SearchResult{
		Contents:   contents,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}
}
