package provider_a

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"search-engine-service/internal/domain"
	"search-engine-service/internal/infra/provider"
)

const testEndpoint = "https://provider-a.example.com/api/contents"

func newTestClient() *Client {
	cfg := provider.ClientConfig{
		BaseURL: "https://provider-a.example.com",
		Timeout: 5 * time.Second,
		Retry: provider.RetryConfig{
			MaxAttempts: 3,
			WaitTime:    100 * time.Millisecond,
			MaxWaitTime: 500 * time.Millisecond,
		},
		CB: provider.CBConfig{
			MaxRequests:  5,
			Interval:     60 * time.Second,
			Timeout:      15 * time.Second,
			FailureRatio: 0.6,
		},
	}
	logger := zap.NewNop()
	client := New(cfg, logger)

	// Activate httpmock for this client's HTTP transport
	httpmock.ActivateNonDefault(client.client.GetClient())

	return client
}

func mockSuccessResponse() Response {
	return Response{
		Contents: []ContentItem{
			{
				ID:    "video-1",
				Title: "Test Video 1",
				Type:  "video",
				Metrics: Metrics{
					Views:    10000,
					Likes:    500,
					Duration: "5m30s",
				},
				PublishedAt: "2024-01-15T10:00:00Z",
				Tags:        []string{"golang", "tutorial"},
			},
			{
				ID:    "video-2",
				Title: "Test Video 2",
				Type:  "video",
				Metrics: Metrics{
					Views:    5000,
					Likes:    250,
					Duration: "3m45s",
				},
				PublishedAt: "2024-01-16T12:00:00Z",
				Tags:        []string{"testing"},
			},
		},
		Pagination: Pagination{
			Total:   2,
			Page:    1,
			PerPage: 10,
		},
	}
}

// TestProviderA_Fetch_Success tests successful JSON fetch and parse.
func TestProviderA_Fetch_Success(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	mockResp := mockSuccessResponse()
	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewJsonResponderOrPanic(200, mockResp))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.NoError(t, err)
	assert.Len(t, contents, 2)

	// Verify first content
	assert.Equal(t, "provider_a", contents[0].ProviderID)
	assert.Equal(t, "video-1", contents[0].ExternalID)
	assert.Equal(t, "Test Video 1", contents[0].Title)
	assert.Equal(t, domain.ContentTypeVideo, contents[0].Type)
	assert.Equal(t, 10000, contents[0].Views)
	assert.Equal(t, 500, contents[0].Likes)
	assert.Equal(t, "5m30s", contents[0].Duration)
	assert.Equal(t, []string{"golang", "tutorial"}, contents[0].Tags)
	assert.Greater(t, contents[0].Score, 0.0) // Score should be calculated

	// Verify second content
	assert.Equal(t, "video-2", contents[1].ExternalID)
	assert.Equal(t, "Test Video 2", contents[1].Title)
}

// TestProviderA_Fetch_EmptyResponse tests handling of empty content array.
func TestProviderA_Fetch_EmptyResponse(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	emptyResp := Response{
		Contents: []ContentItem{},
		Pagination: Pagination{
			Total:   0,
			Page:    1,
			PerPage: 10,
		},
	}

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewJsonResponderOrPanic(200, emptyResp))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.NoError(t, err)
	assert.Empty(t, contents)
}

// TestProviderA_Fetch_HTTPError_4xx tests client error handling (4xx).
func TestProviderA_Fetch_HTTPError_4xx(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name       string
		statusCode int
	}{
		{"400 Bad Request", 400},
		{"404 Not Found", 404},
		{"429 Too Many Requests", 429},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()
			httpmock.RegisterResponder("GET", testEndpoint,
				httpmock.NewStringResponder(tt.statusCode, "Error"))

			client := newTestClient()
			contents, err := client.Fetch(context.Background())

			require.Error(t, err)
			assert.Nil(t, contents)
			assert.Contains(t, err.Error(), fmt.Sprintf("status %d", tt.statusCode))
		})
	}
}

// TestProviderA_Fetch_HTTPError_5xx tests server error handling (5xx).
func TestProviderA_Fetch_HTTPError_5xx(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name       string
		statusCode int
	}{
		{"500 Internal Server Error", 500},
		{"502 Bad Gateway", 502},
		{"503 Service Unavailable", 503},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()
			httpmock.RegisterResponder("GET", testEndpoint,
				httpmock.NewStringResponder(tt.statusCode, "Server Error"))

			client := newTestClient()
			contents, err := client.Fetch(context.Background())

			require.Error(t, err)
			assert.Nil(t, contents)
			assert.Contains(t, err.Error(), fmt.Sprintf("status %d", tt.statusCode))
		})
	}
}

// TestProviderA_Fetch_NetworkError tests network error handling.
func TestProviderA_Fetch_NetworkError(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewErrorResponder(fmt.Errorf("network error: connection refused")))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.Error(t, err)
	assert.Nil(t, contents)
	assert.Contains(t, err.Error(), "fetching from provider_a")
}

// TestProviderA_Fetch_ContextCancellation tests context cancellation handling.
func TestProviderA_Fetch_ContextCancellation(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	// Mock a slow response
	httpmock.RegisterResponder("GET", testEndpoint,
		func(_ *http.Request) (*http.Response, error) {
			time.Sleep(200 * time.Millisecond)

			return httpmock.NewJsonResponse(200, mockSuccessResponse())
		})

	client := newTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	contents, err := client.Fetch(ctx)

	require.Error(t, err)
	assert.Nil(t, contents)
}

// TestProviderA_CircuitBreaker_Opens tests that CB opens after consecutive failures.
func TestProviderA_CircuitBreaker_Opens(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	// Mock 500 errors
	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewStringResponder(500, "Internal Server Error"))

	client := newTestClient()

	// Trigger consecutive failures - CB needs FailureRatio >= 0.6 with min 3 requests
	// So we need at least 3 requests with 60% failure rate
	for i := 0; i < 5; i++ {
		_, err := client.Fetch(context.Background())
		require.Error(t, err)
	}

	// CB should be open now - next request should fail immediately
	start := time.Now()
	_, err := client.Fetch(context.Background())
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
	// Should fail fast (< 100ms) without making HTTP request
	assert.Less(t, elapsed.Milliseconds(), int64(100))
}

// TestProviderA_Retry_ExponentialBackoff tests retry mechanism.
func TestProviderA_Retry_ExponentialBackoff(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	callCount := 0
	httpmock.RegisterResponder("GET", testEndpoint,
		func(_ *http.Request) (*http.Response, error) {
			callCount++
			if callCount < 3 {
				// Fail first 2 attempts
				return httpmock.NewStringResponse(500, "Server Error"), nil
			}
			// Succeed on 3rd attempt
			return httpmock.NewJsonResponse(200, mockSuccessResponse())
		})

	start := time.Now()
	client := newTestClient()
	contents, err := client.Fetch(context.Background())
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, contents, 2)
	assert.Equal(t, 3, callCount, "Should retry twice and succeed on 3rd attempt")

	// With exponential backoff: wait1=100ms, wait2=200ms → total >= 300ms
	// We use 100ms as base for faster tests
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(200))
}

// TestProviderA_Retry_MaxRetriesExceeded tests behavior when all retries fail.
func TestProviderA_Retry_MaxRetriesExceeded(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	callCount := 0
	httpmock.RegisterResponder("GET", testEndpoint,
		func(_ *http.Request) (*http.Response, error) {
			callCount++

			return httpmock.NewStringResponse(500, "Server Error"), nil
		})

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.Error(t, err)
	assert.Nil(t, contents)
	// Should make 1 initial request + 3 retries = 4 total calls
	assert.Equal(t, 4, callCount)
}

// TestProviderA_Name tests the Name method.
func TestProviderA_Name(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	client := newTestClient()
	assert.Equal(t, "provider_a", client.Name())
}

// TestProviderA_Fetch_ScoreCalculation verifies score is calculated for each content.
func TestProviderA_Fetch_ScoreCalculation(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewJsonResponderOrPanic(200, mockSuccessResponse()))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.NoError(t, err)
	for _, content := range contents {
		assert.Greater(t, content.Score, 0.0, "Score should be calculated and positive")
	}
}

// TestProviderA_Fetch_DateParsing tests published_at date parsing.
func TestProviderA_Fetch_DateParsing(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	resp := Response{
		Contents: []ContentItem{
			{
				ID:          "video-1",
				Title:       "Test",
				Type:        "video",
				Metrics:     Metrics{Views: 100, Likes: 10},
				PublishedAt: "2024-01-15T10:30:00Z",
				Tags:        []string{},
			},
		},
	}

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewJsonResponderOrPanic(200, resp))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.NoError(t, err)
	require.Len(t, contents, 1)

	expectedTime, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
	assert.Equal(t, expectedTime, contents[0].PublishedAt)
}

// TestProviderA_Fetch_InvalidDateFormat tests handling of invalid date format.
func TestProviderA_Fetch_InvalidDateFormat(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	resp := Response{
		Contents: []ContentItem{
			{
				ID:          "video-1",
				Title:       "Test",
				Type:        "video",
				Metrics:     Metrics{Views: 100, Likes: 10},
				PublishedAt: "invalid-date", // Invalid date format
				Tags:        []string{},
			},
		},
	}

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewJsonResponderOrPanic(200, resp))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	// Should still succeed but with zero time
	require.NoError(t, err)
	require.Len(t, contents, 1)
	assert.True(t, contents[0].PublishedAt.IsZero())
}

// TestProviderA_Fetch_HTTPCallCount verifies httpmock call tracking.
func TestProviderA_Fetch_HTTPCallCount(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewJsonResponderOrPanic(200, mockSuccessResponse()))

	client := newTestClient()
	_, err := client.Fetch(context.Background())

	require.NoError(t, err)
	info := httpmock.GetCallCountInfo()
	assert.Equal(t, 1, info["GET "+testEndpoint])
}
