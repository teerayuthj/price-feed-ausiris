package redis

import (
	"context"
	"encoding/json"
	"time"

	"gold-socket/pkg/models"
)

const (
	// Cache keys
	KeyUSDRate    = "cache:usd_rate"
	KeyMarketData = "cache:market_data"

	// Default cache TTL
	DefaultCacheTTL = 5 * time.Minute
)

// SetUSDRate caches USD rate data
func (c *Client) SetUSDRate(ctx context.Context, rate *models.USDRateWithStatus) error {
	data, err := json.Marshal(rate)
	if err != nil {
		return err
	}
	return c.Set(ctx, KeyUSDRate, data, DefaultCacheTTL)
}

// GetUSDRate retrieves cached USD rate data
func (c *Client) GetUSDRate(ctx context.Context) (*models.USDRateWithStatus, error) {
	data, err := c.GetBytes(ctx, KeyUSDRate)
	if err != nil {
		return nil, err
	}

	var rate models.USDRateWithStatus
	if err := json.Unmarshal(data, &rate); err != nil {
		return nil, err
	}

	return &rate, nil
}

// SetMarketData caches market data
func (c *Client) SetMarketData(ctx context.Context, data *models.MarketData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.Set(ctx, KeyMarketData, jsonData, DefaultCacheTTL)
}

// GetMarketData retrieves cached market data
func (c *Client) GetMarketData(ctx context.Context) (*models.MarketData, error) {
	data, err := c.GetBytes(ctx, KeyMarketData)
	if err != nil {
		return nil, err
	}

	var marketData models.MarketData
	if err := json.Unmarshal(data, &marketData); err != nil {
		return nil, err
	}

	return &marketData, nil
}

// InvalidateCache removes all cached data
func (c *Client) InvalidateCache(ctx context.Context) error {
	if err := c.Delete(ctx, KeyUSDRate); err != nil {
		return err
	}
	return c.Delete(ctx, KeyMarketData)
}
