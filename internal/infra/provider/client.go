// Package provider provides HTTP client utilities for external providers.
package provider

import (
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sony/gobreaker/v2"
)

// ClientConfig holds configuration for a provider client.
type ClientConfig struct {
	BaseURL string
	Timeout time.Duration
	Retry   RetryConfig
	CB      CBConfig
}

// RetryConfig holds retry configuration.
type RetryConfig struct {
	MaxAttempts int
	WaitTime    time.Duration
	MaxWaitTime time.Duration
}

// CBConfig holds circuit breaker configuration.
type CBConfig struct {
	MaxRequests  uint32
	Interval     time.Duration
	Timeout      time.Duration
	FailureRatio float64
}

// NewRestyClient creates a new Resty HTTP client with retry configuration.
func NewRestyClient(cfg ClientConfig) *resty.Client {
	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetTimeout(cfg.Timeout).
		SetRetryCount(cfg.Retry.MaxAttempts).
		SetRetryWaitTime(cfg.Retry.WaitTime).
		SetRetryMaxWaitTime(cfg.Retry.MaxWaitTime).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			// Retry on network errors or 5xx status codes
			if err != nil {
				return true
			}

			return r.StatusCode() >= 500
		})

	return client
}

// NewCircuitBreaker creates a new circuit breaker for a provider.
func NewCircuitBreaker[T any](name string, cfg CBConfig) *gobreaker.CircuitBreaker[T] {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)

			return counts.Requests >= 3 && failureRatio >= cfg.FailureRatio
		},
		OnStateChange: func(_ string, _ gobreaker.State, _ gobreaker.State) {
			// Log state changes - logger injected at higher level #todo
		},
	}

	return gobreaker.NewCircuitBreaker[T](settings)
}
