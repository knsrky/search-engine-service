package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"search-engine-service/internal/domain"
	"search-engine-service/internal/validator"
)

func newTestValidator() *validator.Validator {
	return validator.New()
}

// validBaseRequest returns a SearchRequest with valid Page and PageSize
// for tests that focus on other fields.
func validBaseRequest() SearchRequest {
	return SearchRequest{Page: 1, PageSize: 20}
}

// TestSearchRequest_Validation_Valid tests valid search requests.
func TestSearchRequest_Validation_Valid(t *testing.T) {
	v := newTestValidator()

	tests := []struct {
		name string
		req  SearchRequest
	}{
		{
			name: "minimal valid request",
			req:  SearchRequest{Page: 1, PageSize: 1},
		},
		{
			name: "query only",
			req:  SearchRequest{Query: "golang", Page: 1, PageSize: 1},
		},
		{
			name: "full valid request",
			req: SearchRequest{
				Query:     "golang tutorial",
				Type:      "video",
				SortBy:    "score",
				SortOrder: "desc",
				Page:      1,
				PageSize:  20,
			},
		},
		{
			name: "article type",
			req:  SearchRequest{Type: "article", Page: 1, PageSize: 1},
		},
		{
			name: "all sort fields",
			req:  SearchRequest{SortBy: "published_at", Page: 1, PageSize: 1},
		},
		{
			name: "asc sort order",
			req:  SearchRequest{SortOrder: "asc", Page: 1, PageSize: 1},
		},
		{
			name: "max page size",
			req:  SearchRequest{Page: 1, PageSize: 100},
		},
		{
			name: "query at max length",
			req:  SearchRequest{Query: string(make([]byte, 200)), Page: 1, PageSize: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(&tt.req)
			assert.NoError(t, err)
		})
	}
}

// TestSearchRequest_Validation_Invalid tests invalid search requests.
func TestSearchRequest_Validation_Invalid(t *testing.T) {
	v := newTestValidator()

	tests := []struct {
		name         string
		req          SearchRequest
		expectField  string
		expectTag    string
		expectErrMsg string
	}{
		{
			name:         "query too long",
			req:          SearchRequest{Query: string(make([]byte, 201)), Page: 1, PageSize: 1},
			expectField:  "Query",
			expectTag:    "max",
			expectErrMsg: "must be at most 200",
		},
		{
			name:         "invalid type",
			req:          SearchRequest{Type: "podcast", Page: 1, PageSize: 1},
			expectField:  "Type",
			expectTag:    "oneof",
			expectErrMsg: "must be one of: video article",
		},
		{
			name:         "invalid sort field",
			req:          SearchRequest{SortBy: "invalid_field", Page: 1, PageSize: 1},
			expectField:  "SortBy",
			expectTag:    "oneof",
			expectErrMsg: "must be one of: relevance score published_at",
		},
		{
			name:         "invalid sort order",
			req:          SearchRequest{SortOrder: "random", Page: 1, PageSize: 1},
			expectField:  "SortOrder",
			expectTag:    "oneof",
			expectErrMsg: "must be one of: asc desc",
		},
		{
			name:         "negative page",
			req:          SearchRequest{Page: -1, PageSize: 1},
			expectField:  "Page",
			expectTag:    "min",
			expectErrMsg: "must be at least 1",
		},
		{
			name:         "page size too large",
			req:          SearchRequest{Page: 1, PageSize: 101},
			expectField:  "PageSize",
			expectTag:    "max",
			expectErrMsg: "must be at most 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(&tt.req)
			require.Error(t, err)

			// Check that error is ValidationErrors
			validationErrs, ok := err.(validator.ValidationErrors)
			require.True(t, ok, "expected ValidationErrors type")
			require.NotEmpty(t, validationErrs)

			// Find the expected field error
			found := false
			for _, ve := range validationErrs {
				if ve.Field == tt.expectField {
					found = true
					assert.Equal(t, tt.expectTag, ve.Tag)
					assert.Contains(t, ve.Message, tt.expectErrMsg)
				}
			}
			assert.True(t, found, "expected error for field %s", tt.expectField)
		})
	}
}

// TestSearchRequest_Validation_MultipleErrors tests requests with multiple validation errors.
func TestSearchRequest_Validation_MultipleErrors(t *testing.T) {
	v := newTestValidator()

	req := SearchRequest{
		Query:     string(make([]byte, 201)), // too long
		Type:      "invalid",                 // invalid type
		SortBy:    "invalid_sort",            // invalid sort field
		SortOrder: "invalid_order",           // invalid sort order
		Page:      0,                         // invalid page
		PageSize:  200,                       // too large
	}

	err := v.Validate(&req)
	require.Error(t, err)

	validationErrs, ok := err.(validator.ValidationErrors)
	require.True(t, ok)

	// Should have multiple errors
	assert.GreaterOrEqual(t, len(validationErrs), 5)

	// Verify Error() method produces concatenated message
	errMsg := validationErrs.Error()
	assert.Contains(t, errMsg, "Query")
	assert.Contains(t, errMsg, "Type")
}

// TestSearchRequest_ToSearchParams tests conversion to domain SearchParams.
func TestSearchRequest_ToSearchParams(t *testing.T) {
	tests := []struct {
		name     string
		req      SearchRequest
		expected domain.SearchParams
	}{
		{
			name: "empty request uses defaults",
			req:  SearchRequest{},
			expected: domain.SearchParams{
				Query:     "",
				Type:      "",
				SortBy:    domain.SortFieldScore,
				SortOrder: domain.SortOrderDesc,
				Page:      1,
				PageSize:  5,
			},
		},
		{
			name: "full request converts correctly",
			req: SearchRequest{
				Query:     "golang",
				Type:      "video",
				SortBy:    "published_at",
				SortOrder: "asc",
				Page:      3,
				PageSize:  50,
			},
			expected: domain.SearchParams{
				Query:     "golang",
				Type:      domain.ContentTypeVideo,
				SortBy:    domain.SortFieldPublishedAt,
				SortOrder: domain.SortOrderAsc,
				Page:      3,
				PageSize:  50,
			},
		},
		{
			name: "query without sort_by defaults to relevance",
			req:  SearchRequest{Query: "go"},
			expected: domain.SearchParams{
				Query:     "go",
				Type:      "",
				SortBy:    domain.SortFieldRelevance,
				SortOrder: domain.SortOrderDesc,
				Page:      1,
				PageSize:  5,
			},
		},
		{
			name: "query with explicit sort_by uses specified sort",
			req:  SearchRequest{Query: "go", SortBy: "score"},
			expected: domain.SearchParams{
				Query:     "go",
				Type:      "",
				SortBy:    domain.SortFieldScore,
				SortOrder: domain.SortOrderDesc,
				Page:      1,
				PageSize:  5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.req.ToSearchParams()

			assert.Equal(t, tt.expected.Query, result.Query)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.SortBy, result.SortBy)
			assert.Equal(t, tt.expected.SortOrder, result.SortOrder)
			assert.Equal(t, tt.expected.Page, result.Page)
			assert.Equal(t, tt.expected.PageSize, result.PageSize)
		})
	}
}

// TestSyncRequest_Validation tests SyncRequest validation.
func TestSyncRequest_Validation(t *testing.T) {
	v := newTestValidator()

	tests := []struct {
		name    string
		req     SyncRequest
		wantErr bool
	}{
		{
			name:    "empty request (valid)",
			req:     SyncRequest{},
			wantErr: false,
		},
		{
			name:    "valid provider",
			req:     SyncRequest{Provider: "provider_a"},
			wantErr: false,
		},
		{
			name:    "provider too long",
			req:     SyncRequest{Provider: string(make([]byte, 51))},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(&tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidationErrors_Error tests the Error() method of ValidationErrors.
func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name     string
		errs     validator.ValidationErrors
		expected string
	}{
		{
			name:     "empty errors",
			errs:     validator.ValidationErrors{},
			expected: "",
		},
		{
			name: "single error",
			errs: validator.ValidationErrors{
				{Field: "Query", Message: "Query is required"},
			},
			expected: "Query is required",
		},
		{
			name: "multiple errors",
			errs: validator.ValidationErrors{
				{Field: "Query", Message: "Query is required"},
				{Field: "Page", Message: "Page must be at least 1"},
			},
			expected: "Query is required; Page must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.errs.Error())
		})
	}
}

// TestSearchRequest_Validation_ContentTypes tests all content type variations.
func TestSearchRequest_Validation_ContentTypes(t *testing.T) {
	v := newTestValidator()

	validTypes := []string{"", "video", "article"}
	invalidTypes := []string{"text", "podcast", "image", "VIDEO", "Article"}

	for _, contentType := range validTypes {
		t.Run("valid_"+contentType, func(t *testing.T) {
			req := validBaseRequest()
			req.Type = contentType
			err := v.Validate(&req)
			assert.NoError(t, err)
		})
	}

	for _, contentType := range invalidTypes {
		t.Run("invalid_"+contentType, func(t *testing.T) {
			req := validBaseRequest()
			req.Type = contentType
			err := v.Validate(&req)
			assert.Error(t, err)
		})
	}
}

// TestSearchRequest_Validation_SortFields tests all sort field variations.
func TestSearchRequest_Validation_SortFields(t *testing.T) {
	v := newTestValidator()

	validFields := []string{"", "relevance", "score", "published_at"}
	invalidFields := []string{"date", "created_at", "SCORE", "invalid", "views", "likes", "title"}

	for _, sortField := range validFields {
		t.Run("valid_"+sortField, func(t *testing.T) {
			req := validBaseRequest()
			req.SortBy = sortField
			err := v.Validate(&req)
			assert.NoError(t, err)
		})
	}

	for _, sortField := range invalidFields {
		t.Run("invalid_"+sortField, func(t *testing.T) {
			req := validBaseRequest()
			req.SortBy = sortField
			err := v.Validate(&req)
			assert.Error(t, err)
		})
	}
}

// TestSearchRequest_Validation_SortOrders tests all sort order variations.
func TestSearchRequest_Validation_SortOrders(t *testing.T) {
	v := newTestValidator()

	validOrders := []string{"", "asc", "desc"}
	invalidOrders := []string{"ascending", "descending", "ASC", "DESC"}

	for _, sortOrder := range validOrders {
		t.Run("valid_"+sortOrder, func(t *testing.T) {
			req := validBaseRequest()
			req.SortOrder = sortOrder
			err := v.Validate(&req)
			assert.NoError(t, err)
		})
	}

	for _, sortOrder := range invalidOrders {
		t.Run("invalid_"+sortOrder, func(t *testing.T) {
			req := validBaseRequest()
			req.SortOrder = sortOrder
			err := v.Validate(&req)
			assert.Error(t, err)
		})
	}
}
