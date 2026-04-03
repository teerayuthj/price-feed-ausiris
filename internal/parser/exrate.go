package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gold-socket/pkg/models"
)

// GetLatestUSDRate reads the last line from exrate.txt and returns USD rate
func GetLatestUSDRate(filePath string) (*models.USDRate, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var lastLine string
	scanner := bufio.NewScanner(file)

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

	// Parse: "HH:MM:SSUSD                     buy_rate    sell_rate"
	parts := strings.Fields(lastLine)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid data format in last line: %s", lastLine)
	}

	timeAndCurrency := parts[0]
	if len(timeAndCurrency) < 11 || !strings.HasSuffix(timeAndCurrency, "USD") {
		return nil, fmt.Errorf("invalid time/currency format: %s", timeAndCurrency)
	}

	timeStr := timeAndCurrency[:8]
	currency := timeAndCurrency[8:]

	buyRate, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid buy rate: %v", err)
	}

	sellRate, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid sell rate: %v", err)
	}

	return &models.USDRate{
		Timestamp: NowInBangkok(),
		Time:      timeStr,
		Currency:  currency,
		Buy:       RoundToTwoDecimals(buyRate),
		Sell:      RoundToTwoDecimals(sellRate),
	}, nil
}

// LoadLatestUSDRate loads the latest USD rate from the default file
func LoadLatestUSDRate() (*models.USDRate, error) {
	return GetLatestUSDRate(ExrateFilePath())
}

// SaveUSDRateToJSON saves USD rate to JSON file
func SaveUSDRateToJSON(rate *models.USDRate, isManual bool) error {
	source := SourceAuto
	if isManual {
		source = SourceManual
	}

	usdRateWithStatus := &models.USDRateWithStatus{
		Timestamp:    rate.Timestamp,
		Time:         rate.Time,
		Currency:     rate.Currency,
		Buy:          rate.Buy,
		Sell:         rate.Sell,
		MarketStatus: GetMarketStatus(),
		Source:       source,
	}

	jsonData, err := json.MarshalIndent(usdRateWithStatus, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal USD rate: %v", err)
	}

	usdJSONFile := USDJSONFilePath()
	if err := os.MkdirAll(filepath.Dir(usdJSONFile), 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	err = os.WriteFile(usdJSONFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file: %v", err)
	}

	return nil
}

// LoadUSDRateFromJSON loads USD rate from JSON file
func LoadUSDRateFromJSON() (*models.USDRateWithStatus, error) {
	usdJSONFile := USDJSONFilePath()
	if _, err := os.Stat(usdJSONFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("JSON file does not exist")
	}

	data, err := os.ReadFile(usdJSONFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %v", err)
	}

	var rate models.USDRateWithStatus
	err = json.Unmarshal(data, &rate)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return &rate, nil
}

// UpdateUSDRateFromExrate updates JSON file from exrate.txt (latest line)
func UpdateUSDRateFromExrate() error {
	rate, err := LoadLatestUSDRate()
	if err != nil {
		return fmt.Errorf("failed to load latest USD rate: %v", err)
	}

	return SaveUSDRateToJSON(rate, false)
}

// GetCurrentUSDRate gets USD rate from JSON file or exrate.txt as fallback
func GetCurrentUSDRate() (*models.USDRateWithStatus, error) {
	rate, err := LoadUSDRateFromJSON()
	if err == nil {
		return rate, nil
	}

	exrateData, err := LoadLatestUSDRate()
	if err != nil {
		return nil, fmt.Errorf("failed to load from both JSON and exrate.txt: %v", err)
	}

	rate = &models.USDRateWithStatus{
		Timestamp:    exrateData.Timestamp,
		Time:         exrateData.Time,
		Currency:     exrateData.Currency,
		Buy:          exrateData.Buy,
		Sell:         exrateData.Sell,
		MarketStatus: GetMarketStatus(),
		Source:       SourceAuto,
	}

	SaveUSDRateToJSON(exrateData, false)

	return rate, nil
}

// CreateManualUSDRate creates a manual USD rate entry
func CreateManualUSDRate(buy, sell float64) error {
	now := NowInBangkok()

	rate := &models.USDRate{
		Timestamp: now,
		Time:      now.Format("15:04:05"),
		Currency:  "USD",
		Buy:       RoundToTwoDecimals(buy),
		Sell:      RoundToTwoDecimals(sell),
	}

	return SaveUSDRateToJSON(rate, true)
}
