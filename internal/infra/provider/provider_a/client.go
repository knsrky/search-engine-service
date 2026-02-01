// Package provider_a implements the JSON content provider client.
package provider_a

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/sony/gobreaker/v2"
	"go.uber.org/zap"

	"search-engine-service/internal/domain"
	"search-engine-service/internal/infra/provider"
)

// Endpoint is the API path for Provider A's content endpoint.
const Endpoint = "/api/contents"

// Client implements domain.Provider for Provider A (JSON).
type Client struct {
	name   string
	client *resty.Client
	cb     *gobreaker.CircuitBreaker[*resty.Response]
	logger *zap.Logger
}

// New creates a new Provider A client.
func New(cfg provider.ClientConfig, logger *zap.Logger) *Client {
	return &Client{
		name:   "provider_a",
		client: provider.NewRestyClient(cfg),
		cb:     provider.NewCircuitBreaker[*resty.Response]("provider_a", cfg.CB),
		logger: logger,
	}
}

// Name returns the provider identifier.
func (c *Client) Name() string {
	return c.name
}

// Fetch retrieves all content from Provider A.
func (c *Client) Fetch(ctx context.Context) ([]*domain.Content, error) {
	resp, err := c.cb.Execute(func() (*resty.Response, error) {
		var result Response
		r, err := c.client.R().
			SetContext(ctx).
			SetResult(&result).
			Get(Endpoint)
		if err != nil {
			return nil, err
		}
		if r.IsError() {
			return nil, fmt.Errorf("provider_a returned status %d", r.StatusCode())
		}

		return r, nil
	})

	if err != nil {
		c.logger.Warn("provider_a fetch failed",
			zap.Error(err),
			zap.String("state", c.cb.State().String()),
		)

		return nil, fmt.Errorf("fetching from provider_a: %w", err)
	}

	// Parse response
	result := resp.Result().(*Response)
	contents := make([]*domain.Content, 0, len(result.Contents))

	for _, item := range result.Contents {
		content := item.ToDomain(c.name)
		// Calculate score
		content.Score = domain.CalculateScore(content)
		contents = append(contents, content)
	}

	c.logger.Info("provider_a fetch completed",
		zap.Int("count", len(contents)),
	)

	return contents, nil
}

// HealthCheck verifies the provider is accessible.
func (c *Client) HealthCheck(ctx context.Context) error {
	resp, err := c.client.R().
		SetContext(ctx).
		Get("/health")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("health check returned status %d", resp.StatusCode())
	}

	return nil
}
