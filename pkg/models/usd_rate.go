package models

import "time"

// USDRate represents the latest USD exchange rate with 2 decimal places
type USDRate struct {
	Timestamp time.Time `json:"timestamp"`
	Time      string    `json:"time"`
	Currency  string    `json:"currency"`
	Buy       float64   `json:"buy"`
	Sell      float64   `json:"sell"`
}

// USDRateWithStatus represents USD rate with market status
type USDRateWithStatus struct {
	Timestamp    time.Time `json:"timestamp"`
	Time         string    `json:"time"`
	Currency     string    `json:"currency"`
	Buy          float64   `json:"buy"`
	Sell         float64   `json:"sell"`
	MarketStatus string    `json:"market_status"` // "open", "closed"
	Source       string    `json:"source"`        // "auto", "manual"
}
