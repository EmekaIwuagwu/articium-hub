package evm

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// HealthChecker monitors the health of an EVM client
type HealthChecker struct {
	client        *Client
	logger        zerolog.Logger
	isHealthy     bool
	lastCheck     time.Time
	checkInterval time.Duration
	mu            sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(client *Client, logger zerolog.Logger) *HealthChecker {
	hc := &HealthChecker{
		client:        client,
		logger:        logger.With().Str("component", "health_checker").Logger(),
		isHealthy:     true,
		checkInterval: 30 * time.Second,
	}

	// Start background health checking
	go hc.start()

	return hc
}

// start begins periodic health checks
func (hc *HealthChecker) start() {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		healthy := hc.performHealthCheck(ctx)
		cancel()

		hc.mu.Lock()
		hc.isHealthy = healthy
		hc.lastCheck = time.Now()
		hc.mu.Unlock()

		if !healthy {
			hc.logger.Warn().Msg("Health check failed")
		}
	}
}

// performHealthCheck performs a health check
func (hc *HealthChecker) performHealthCheck(ctx context.Context) bool {
	// Try to get latest block number
	_, err := hc.client.GetLatestBlockNumber(ctx)
	if err != nil {
		hc.logger.Warn().Err(err).Msg("Failed to get latest block number")
		return false
	}

	// Verify chain ID
	chainID, err := hc.client.ChainID(ctx)
	if err != nil {
		hc.logger.Warn().Err(err).Msg("Failed to get chain ID")
		return false
	}

	expectedChainID := hc.client.config.ChainID
	if chainID.String() != expectedChainID {
		hc.logger.Warn().
			Str("expected", expectedChainID).
			Str("actual", chainID.String()).
			Msg("Chain ID mismatch")
		return false
	}

	return true
}

// IsHealthy returns the current health status
func (hc *HealthChecker) IsHealthy(ctx context.Context) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.isHealthy
}

// GetLastCheckTime returns the time of the last health check
func (hc *HealthChecker) GetLastCheckTime() time.Time {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.lastCheck
}
