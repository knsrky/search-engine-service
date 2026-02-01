package registry

import (
	"search-engine-service/internal/config"
	"search-engine-service/internal/domain"
	"search-engine-service/internal/infra/provider"
	"search-engine-service/internal/infra/provider/provider_a"
	"search-engine-service/internal/infra/provider/provider_b"

	"go.uber.org/zap"
)

// NewProviders creates all configured provider clients.
// This is a factory function that centralizes provider initialization
// while maintaining dependency injection principles.
//
// Parameters:
//   - cfg: Provider configuration containing endpoints, timeouts, retry, and circuit breaker settings
//   - logger: Zap logger instance for structured logging
//
// Returns a slice of domain.Provider instances ready for use in services.
func NewProviders(cfg config.ProviderConfig, logger *zap.Logger) []domain.Provider {
	providers := make([]domain.Provider, 0, 2)

	// Provider A
	providerA := provider_a.New(
		provider.ClientConfig{
			BaseURL: cfg.A.BaseURL,
			Timeout: cfg.A.Timeout,
			Retry: provider.RetryConfig{
				MaxAttempts: cfg.A.Retry.MaxAttempts,
				WaitTime:    cfg.A.Retry.WaitTime,
				MaxWaitTime: cfg.A.Retry.MaxWaitTime,
			},
			CB: provider.CBConfig{
				MaxRequests:  cfg.A.CB.MaxRequests,
				Interval:     cfg.A.CB.Interval,
				Timeout:      cfg.A.CB.Timeout,
				FailureRatio: cfg.A.CB.FailureRatio,
			},
		},
		logger,
	)
	providers = append(providers, providerA)

	// Provider B
	providerB := provider_b.New(
		provider.ClientConfig{
			BaseURL: cfg.B.BaseURL,
			Timeout: cfg.B.Timeout,
			Retry: provider.RetryConfig{
				MaxAttempts: cfg.B.Retry.MaxAttempts,
				WaitTime:    cfg.B.Retry.WaitTime,
				MaxWaitTime: cfg.B.Retry.MaxWaitTime,
			},
			CB: provider.CBConfig{
				MaxRequests:  cfg.B.CB.MaxRequests,
				Interval:     cfg.B.CB.Interval,
				Timeout:      cfg.B.CB.Timeout,
				FailureRatio: cfg.B.CB.FailureRatio,
			},
		},
		logger,
	)
	providers = append(providers, providerB)

	return providers
}
