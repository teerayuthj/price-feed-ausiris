package parser

import (
	"math"
	"time"
)

// MarketStatus constants
const (
	MarketStatusOpen   = "open"
	MarketStatusClosed = "closed"
	SourceAuto         = "auto"
	SourceManual       = "manual"
)

// RoundToTwoDecimals rounds a float64 to 2 decimal places
func RoundToTwoDecimals(value float64) float64 {
	return math.Round(value*100) / 100
}

// IsMarketOpen checks if market is open based on time (Thai timezone)
func IsMarketOpen() bool {
	now := time.Now()
	hour := now.Hour()
	dayOfWeek := now.Weekday()

	// Market closed on weekends
	if dayOfWeek == time.Saturday || dayOfWeek == time.Sunday {
		return false
	}

	// Market hours: 9:00 AM - 5:00 PM (17:00)
	if hour < 9 || hour >= 17 {
		return false
	}

	// Market closed during lunch: 12:00 PM - 1:00 PM
	if hour == 12 {
		return false
	}

	return true
}

// GetMarketStatus returns current market status string
func GetMarketStatus() string {
	if IsMarketOpen() {
		return MarketStatusOpen
	}
	return MarketStatusClosed
}
