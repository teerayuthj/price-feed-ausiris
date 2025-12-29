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

var bangkokLocation = loadBangkokLocation()

func loadBangkokLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return time.Local
	}
	return location
}

// NowInBangkok returns current time in Bangkok timezone.
func NowInBangkok() time.Time {
	return time.Now().In(bangkokLocation)
}

// RoundToTwoDecimals rounds a float64 to 2 decimal places
func RoundToTwoDecimals(value float64) float64 {
	return math.Round(value*100) / 100
}

// IsMarketOpen checks if market is open based on time (Thai timezone)
// Market hours: Monday 07:00 - Saturday 04:00 (21 hours/day)
func IsMarketOpen() bool {
	now := NowInBangkok()
	hour := now.Hour()
	dayOfWeek := now.Weekday()

	// Sunday: always closed
	if dayOfWeek == time.Sunday {
		return false
	}

	// Saturday: open 00:00-04:00 only (continuation from Friday night)
	if dayOfWeek == time.Saturday {
		return hour < 4
	}

	// Monday: open from 07:00 onwards
	if dayOfWeek == time.Monday {
		return hour >= 7
	}

	// Tuesday-Friday:
	// - 00:00-04:00: open (continuation from previous day)
	// - 04:00-07:00: closed
	// - 07:00-24:00: open
	if hour < 4 {
		return true
	}
	if hour < 7 {
		return false
	}
	return true
}

// GetMarketStatus returns current market status string (time-based only)
func GetMarketStatus() string {
	if IsMarketOpen() {
		return MarketStatusOpen
	}
	return MarketStatusClosed
}

// GetMarketStatusWithData returns market status considering time, source connection, and price validity
// Returns "closed" if: outside market hours, source disconnected, or price is zero/invalid
func GetMarketStatusWithData(sourceConnected bool, priceValid bool) string {
	// If source is disconnected or price is invalid, market is closed
	if !sourceConnected || !priceValid {
		return MarketStatusClosed
	}

	// Check time-based market hours
	if IsMarketOpen() {
		return MarketStatusOpen
	}
	return MarketStatusClosed
}
