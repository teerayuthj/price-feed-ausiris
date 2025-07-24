package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

// USDRateWithStatus represents USD rate with market status
type USDRateWithStatus struct {
	Timestamp    time.Time `json:"timestamp"`
	Time         string    `json:"time"`
	Currency     string    `json:"currency"`
	Buy          float64   `json:"buy"`
	Sell         float64   `json:"sell"`
	MarketStatus string    `json:"market_status"` // "open", "closed", "manual"
	Source       string    `json:"source"`        // "auto", "manual"
}

const (
	USD_JSON_FILE = "raw-data/usd_rate.json"
)

// isMarketOpen checks if market is open based on time (Thai timezone)
func isMarketOpen() bool {
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

// getMarketStatus returns current market status
func getMarketStatus() string {
	if isMarketOpen() {
		return "open"
	}
	return "closed"
}

// SaveUSDRateToJSON saves USD rate to JSON file
func SaveUSDRateToJSON(rate *USDRate, isManual bool) error {
	var source string
	if isManual {
		source = "manual"
	} else {
		source = "auto"
	}

	usdRateWithStatus := &USDRateWithStatus{
		Timestamp:    rate.Timestamp,
		Time:         rate.Time,
		Currency:     rate.Currency,
		Buy:          rate.Buy,
		Sell:         rate.Sell,
		MarketStatus: getMarketStatus(),
		Source:       source,
	}

	jsonData, err := json.MarshalIndent(usdRateWithStatus, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal USD rate: %v", err)
	}

	// Create directory if it doesn't exist
	os.MkdirAll("raw-data", 0755)

	err = ioutil.WriteFile(USD_JSON_FILE, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file: %v", err)
	}

	return nil
}

// LoadUSDRateFromJSON loads USD rate from JSON file
func LoadUSDRateFromJSON() (*USDRateWithStatus, error) {
	// Check if file exists
	if _, err := os.Stat(USD_JSON_FILE); os.IsNotExist(err) {
		return nil, fmt.Errorf("JSON file does not exist")
	}

	data, err := ioutil.ReadFile(USD_JSON_FILE)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %v", err)
	}

	var rate USDRateWithStatus
	err = json.Unmarshal(data, &rate)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return &rate, nil
}

// UpdateUSDRateFromExrate updates JSON file from exrate.txt (latest line)
func UpdateUSDRateFromExrate() error {
	// Get latest rate from exrate.txt
	rate, err := LoadLatestUSDRate()
	if err != nil {
		return fmt.Errorf("failed to load latest USD rate: %v", err)
	}

	// Save to JSON file (auto source)
	return SaveUSDRateToJSON(rate, false)
}

// GetCurrentUSDRate gets USD rate from JSON file or exrate.txt as fallback  
func GetCurrentUSDRate() (*USDRateWithStatus, error) {
	// Try to load from JSON file first
	rate, err := LoadUSDRateFromJSON()
	if err == nil {
		return rate, nil
	}

	// Fallback to exrate.txt
	exrateData, err := LoadLatestUSDRate()
	if err != nil {
		return nil, fmt.Errorf("failed to load from both JSON and exrate.txt: %v", err)
	}

	// Convert to USDRateWithStatus
	rate = &USDRateWithStatus{
		Timestamp:    exrateData.Timestamp,
		Time:         exrateData.Time,
		Currency:     exrateData.Currency,
		Buy:          exrateData.Buy,
		Sell:         exrateData.Sell,
		MarketStatus: getMarketStatus(),
		Source:       "auto",
	}

	// Save to JSON for future use
	SaveUSDRateToJSON(exrateData, false)

	return rate, nil
}

// CreateManualUSDRate creates a manual USD rate entry
func CreateManualUSDRate(buy, sell float64) error {
	now := time.Now()
	
	rate := &USDRate{
		Timestamp: now,
		Time:      now.Format("15:04:05"),
		Currency:  "USD",
		Buy:       roundToTwoDecimals(buy),
		Sell:      roundToTwoDecimals(sell),
	}

	return SaveUSDRateToJSON(rate, true)
}