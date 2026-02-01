package provider_b

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

const testEndpoint = "https://provider-b.example.com/feed"

func newTestClient() *Client {
	cfg := provider.ClientConfig{
		BaseURL: "https://provider-b.example.com",
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

func mockSuccessXMLResponse() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<feed>
	<items>
		<item>
			<id>article-1</id>
			<headline>Test Article 1</headline>
			<type>article</type>
			<stats>
				<reading_time>5</reading_time>
				<reactions>150</reactions>
				<comments>25</comments>
			</stats>
			<publication_date>2024-01-15</publication_date>
			<categories>
				<category>technology</category>
				<category>golang</category>
			</categories>
		</item>
		<item>
			<id>video-1</id>
			<headline>Test Video 1</headline>
			<type>video</type>
			<stats>
				<views>10000</views>
				<likes>500</likes>
				<duration>5m30s</duration>
			</stats>
			<publication_date>2024-01-16</publication_date>
			<categories>
				<category>tutorial</category>
			</categories>
		</item>
	</items>
	<meta>
		<total_count>2</total_count>
		<current_page>1</current_page>
		<items_per_page>10</items_per_page>
	</meta>
</feed>`
}

// TestProviderB_Fetch_Success tests successful XML fetch and parse.
func TestProviderB_Fetch_Success(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	mockXML := mockSuccessXMLResponse()
	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewStringResponder(200, mockXML))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.NoError(t, err)
	assert.Len(t, contents, 2)

	// Verify first content (article)
	assert.Equal(t, "provider_b", contents[0].ProviderID)
	assert.Equal(t, "article-1", contents[0].ExternalID)
	assert.Equal(t, "Test Article 1", contents[0].Title)
	assert.Equal(t, domain.ContentTypeArticle, contents[0].Type)
	assert.Equal(t, 5, contents[0].ReadingTime)
	assert.Equal(t, 150, contents[0].Reactions)
	assert.Equal(t, 25, contents[0].Comments)
	assert.Equal(t, []string{"technology", "golang"}, contents[0].Tags)
	assert.Greater(t, contents[0].Score, 0.0) // Score should be calculated

	// Verify second content (video)
	assert.Equal(t, "video-1", contents[1].ExternalID)
	assert.Equal(t, "Test Video 1", contents[1].Title)
	assert.Equal(t, domain.ContentTypeVideo, contents[1].Type)
	assert.Equal(t, 10000, contents[1].Views)
	assert.Equal(t, 500, contents[1].Likes)
	assert.Equal(t, "5m30s", contents[1].Duration)
}

// TestProviderB_Fetch_EmptyResponse tests handling of empty XML.
func TestProviderB_Fetch_EmptyResponse(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	emptyXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed>
	<items></items>
	<meta>
		<total_count>0</total_count>
		<current_page>1</current_page>
		<items_per_page>10</items_per_page>
	</meta>
</feed>`

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewStringResponder(200, emptyXML))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.NoError(t, err)
	assert.Empty(t, contents)
}

// TestProviderB_Fetch_HTTPError_4xx tests client error handling (4xx).
func TestProviderB_Fetch_HTTPError_4xx(t *testing.T) {
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

// TestProviderB_Fetch_HTTPError_5xx tests server error handling (5xx).
func TestProviderB_Fetch_HTTPError_5xx(t *testing.T) {
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

// TestProviderB_Fetch_InvalidXML tests handling of malformed XML.
func TestProviderB_Fetch_InvalidXML(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewStringResponder(200, "not xml at all"))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.Error(t, err)
	assert.Nil(t, contents)
	assert.Contains(t, err.Error(), "parsing provider_b XML")
}

// TestProviderB_Fetch_NetworkError tests network error handling.
func TestProviderB_Fetch_NetworkError(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewErrorResponder(fmt.Errorf("network error: connection refused")))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.Error(t, err)
	assert.Nil(t, contents)
	assert.Contains(t, err.Error(), "fetching from provider_b")
}

// TestProviderB_Fetch_ContextCancellation tests context cancellation handling.
func TestProviderB_Fetch_ContextCancellation(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	// Mock a slow response
	httpmock.RegisterResponder("GET", testEndpoint,
		func(_ *http.Request) (*http.Response, error) {
			time.Sleep(200 * time.Millisecond)

			return httpmock.NewStringResponse(200, mockSuccessXMLResponse()), nil
		})

	client := newTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	contents, err := client.Fetch(ctx)

	require.Error(t, err)
	assert.Nil(t, contents)
}

// TestProviderB_CircuitBreaker_Opens tests that CB opens after consecutive failures.
func TestProviderB_CircuitBreaker_Opens(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	// Mock 500 errors
	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewStringResponder(500, "Internal Server Error"))

	client := newTestClient()

	// Trigger consecutive failures - CB needs FailureRatio >= 0.6 with min 3 requests
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

// TestProviderB_Retry_ExponentialBackoff tests retry mechanism.
func TestProviderB_Retry_ExponentialBackoff(t *testing.T) {
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
			return httpmock.NewStringResponse(200, mockSuccessXMLResponse()), nil
		})

	start := time.Now()
	client := newTestClient()
	contents, err := client.Fetch(context.Background())
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, contents, 2)
	assert.Equal(t, 3, callCount, "Should retry twice and succeed on 3rd attempt")

	// With exponential backoff: wait1=100ms, wait2=200ms → total >= 300ms
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(200))
}

// TestProviderB_Retry_MaxRetriesExceeded tests behavior when all retries fail.
func TestProviderB_Retry_MaxRetriesExceeded(t *testing.T) {
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

// TestProviderB_Name tests the Name method.
func TestProviderB_Name(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	client := newTestClient()
	assert.Equal(t, "provider_b", client.Name())
}

// TestProviderB_Fetch_ScoreCalculation verifies score is calculated for each content.
func TestProviderB_Fetch_ScoreCalculation(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewStringResponder(200, mockSuccessXMLResponse()))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.NoError(t, err)
	for _, content := range contents {
		assert.Greater(t, content.Score, 0.0, "Score should be calculated and positive")
	}
}

// TestProviderB_Fetch_DateParsing tests publication_date parsing.
func TestProviderB_Fetch_DateParsing(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	xmlResp := `<?xml version="1.0" encoding="UTF-8"?>
<feed>
	<items>
		<item>
			<id>article-1</id>
			<headline>Test</headline>
			<type>article</type>
			<stats>
				<reading_time>5</reading_time>
				<reactions>100</reactions>
			</stats>
			<publication_date>2024-01-15</publication_date>
			<categories></categories>
		</item>
	</items>
</feed>`

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewStringResponder(200, xmlResp))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.NoError(t, err)
	require.Len(t, contents, 1)

	expectedTime, _ := time.Parse("2006-01-02", "2024-01-15")
	assert.Equal(t, expectedTime, contents[0].PublishedAt)
}

// TestProviderB_Fetch_InvalidDateFormat tests handling of invalid date format.
func TestProviderB_Fetch_InvalidDateFormat(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	xmlResp := `<?xml version="1.0" encoding="UTF-8"?>
<feed>
	<items>
		<item>
			<id>article-1</id>
			<headline>Test</headline>
			<type>article</type>
			<stats>
				<reading_time>5</reading_time>
				<reactions>100</reactions>
			</stats>
			<publication_date>invalid-date</publication_date>
			<categories></categories>
		</item>
	</items>
</feed>`

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewStringResponder(200, xmlResp))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	// Should still succeed but with zero time
	require.NoError(t, err)
	require.Len(t, contents, 1)
	assert.True(t, contents[0].PublishedAt.IsZero())
}

// TestProviderB_Fetch_MixedContentTypes tests parsing both article and video types.
func TestProviderB_Fetch_MixedContentTypes(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewStringResponder(200, mockSuccessXMLResponse()))

	client := newTestClient()
	contents, err := client.Fetch(context.Background())

	require.NoError(t, err)
	assert.Len(t, contents, 2)

	// Verify article has article-specific fields populated
	articleContent := contents[0]
	assert.Equal(t, domain.ContentTypeArticle, articleContent.Type)
	assert.Greater(t, articleContent.ReadingTime, 0)
	assert.Greater(t, articleContent.Reactions, 0)
	assert.Equal(t, 0, articleContent.Views) // Video field should be empty
	assert.Equal(t, 0, articleContent.Likes) // Video field should be empty

	// Verify video has video-specific fields populated
	videoContent := contents[1]
	assert.Equal(t, domain.ContentTypeVideo, videoContent.Type)
	assert.Greater(t, videoContent.Views, 0)
	assert.Greater(t, videoContent.Likes, 0)
	assert.NotEmpty(t, videoContent.Duration)
	assert.Equal(t, 0, videoContent.ReadingTime) // Article field should be empty
	assert.Equal(t, 0, videoContent.Reactions)   // Article field should be empty
}

// TestProviderB_Fetch_HTTPCallCount verifies httpmock call tracking.
func TestProviderB_Fetch_HTTPCallCount(t *testing.T) {
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", testEndpoint,
		httpmock.NewStringResponder(200, mockSuccessXMLResponse()))

	client := newTestClient()
	_, err := client.Fetch(context.Background())

	require.NoError(t, err)
	info := httpmock.GetCallCountInfo()
	assert.Equal(t, 1, info["GET "+testEndpoint])
}
