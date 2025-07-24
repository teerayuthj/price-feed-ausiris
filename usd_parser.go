package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

// USDRate represents the latest USD exchange rate with 2 decimal places
type USDRate struct {
	Timestamp time.Time `json:"timestamp"`
	Time      string    `json:"time"`
	Currency  string    `json:"currency"`
	Buy       float64   `json:"buy"`
	Sell      float64   `json:"sell"`
}

// roundToTwoDecimals rounds a float64 to 2 decimal places
func roundToTwoDecimals(value float64) float64 {
	return math.Round(value*100) / 100
}

// GetLatestUSDRate reads the last line from exrate.txt and returns USD rate
func GetLatestUSDRate(filePath string) (*USDRate, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var lastLine string
	scanner := bufio.NewScanner(file)

	// Read all lines to get the last one
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lastLine = line
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	if lastLine == "" {
		return nil, fmt.Errorf("no data found in file")
	}

	// Parse the last line: "HH:MM:SSUSD                     buy_rate    sell_rate"
	parts := strings.Fields(lastLine)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid data format in last line: %s", lastLine)
	}

	timeAndCurrency := parts[0]
	if len(timeAndCurrency) < 11 || !strings.HasSuffix(timeAndCurrency, "USD") {
		return nil, fmt.Errorf("invalid time/currency format: %s", timeAndCurrency)
	}

	timeStr := timeAndCurrency[:8] // Extract HH:MM:SS
	currency := timeAndCurrency[8:] // Extract USD

	buyRate, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid buy rate: %v", err)
	}

	sellRate, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid sell rate: %v", err)
	}

	return &USDRate{
		Timestamp: time.Now(),
		Time:      timeStr,
		Currency:  currency,
		Buy:       roundToTwoDecimals(buyRate),
		Sell:      roundToTwoDecimals(sellRate),
	}, nil
}

// LoadLatestUSDRate loads the latest USD rate from the default file
func LoadLatestUSDRate() (*USDRate, error) {
	return GetLatestUSDRate("raw-data/exrate.txt")
}