// Package provider_b implements the XML content provider client.
package provider_b

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/sony/gobreaker/v2"
	"go.uber.org/zap"

	"search-engine-service/internal/domain"
	"search-engine-service/internal/infra/provider"
)

// Endpoint is the API path for Provider B's content endpoint.
const Endpoint = "/feed"

// Client implements domain.Provider for Provider B (XML).
type Client struct {
	name   string
	client *resty.Client
	cb     *gobreaker.CircuitBreaker[*resty.Response]
	logger *zap.Logger
}

// New creates a new Provider B client.
func New(cfg provider.ClientConfig, logger *zap.Logger) *Client {
	return &Client{
		name:   "provider_b",
		client: provider.NewRestyClient(cfg),
		cb:     provider.NewCircuitBreaker[*resty.Response]("provider_b", cfg.CB),
		logger: logger,
	}
}

// Name returns the provider identifier.
func (c *Client) Name() string {
	return c.name
}

// Fetch retrieves all content from Provider B.
func (c *Client) Fetch(ctx context.Context) ([]*domain.Content, error) {
	resp, err := c.cb.Execute(func() (*resty.Response, error) {
		r, err := c.client.R().
			SetContext(ctx).
			SetHeader("Accept", "application/xml").
			Get(Endpoint)
		if err != nil {
			return nil, err
		}
		if r.IsError() {
			return nil, fmt.Errorf("provider_b returned status %d", r.StatusCode())
		}

		return r, nil
	})

	if err != nil {
		c.logger.Warn("provider_b fetch failed",
			zap.Error(err),
			zap.String("state", c.cb.State().String()),
		)

		return nil, fmt.Errorf("fetching from provider_b: %w", err)
	}

	// Parse XML response
	var feed Feed
	if err := xml.Unmarshal(resp.Body(), &feed); err != nil {
		return nil, fmt.Errorf("parsing provider_b XML: %w", err)
	}

	contents := make([]*domain.Content, 0, len(feed.Items.Items))

	for _, item := range feed.Items.Items {
		content := item.ToDomain(c.name)
		// Calculate score
		content.Score = domain.CalculateScore(content)
		contents = append(contents, content)
	}

	c.logger.Info("provider_b fetch completed",
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
