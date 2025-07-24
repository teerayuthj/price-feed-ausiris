package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ExchangeRate represents USD exchange rate data
type ExchangeRate struct {
	Time     string  `json:"time"`
	Currency string  `json:"currency"`
	Buy      float64 `json:"buy"`
	Sell     float64 `json:"sell"`
}

// GoldPrice represents gold price data
type GoldPrice struct {
	USDRate   float64 `json:"usd_rate_buy"`
	USDRateSell float64 `json:"usd_rate_sell"`
	Type      string  `json:"type"`
	BuyBar    float64 `json:"buy_bar"`
	SellBar   float64 `json:"sell_bar"`
	BuyOrnament float64 `json:"buy_ornament"`
	SellOrnament float64 `json:"sell_ornament"`
}

// CombinedData represents the combined JSON structure
type CombinedData struct {
	Timestamp    time.Time      `json:"timestamp"`
	ExchangeRates []ExchangeRate `json:"exchange_rates"`
	GoldPrices   []GoldPrice    `json:"gold_prices"`
}

// ParseExchangeRates parses the exrate.txt file
func ParseExchangeRates(filePath string) ([]ExchangeRate, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var rates []ExchangeRate
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse format: "HH:MM:SSUSD                     buy_rate    sell_rate"
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		timeAndCurrency := parts[0]
		if len(timeAndCurrency) < 11 || !strings.HasSuffix(timeAndCurrency, "USD") {
			continue
		}

		timeStr := timeAndCurrency[:8] // Extract HH:MM:SS
		currency := timeAndCurrency[8:] // Extract USD

		buyRate, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}

		sellRate, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			continue
		}

		rates = append(rates, ExchangeRate{
			Time:     timeStr,
			Currency: currency,
			Buy:      buyRate,
			Sell:     sellRate,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	return rates, nil
}

// ParseGoldPrices parses the market_retail.txt file
func ParseGoldPrices(filePath string) ([]GoldPrice, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var prices []GoldPrice
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse pipe-separated values
		parts := strings.Split(line, "|")
		if len(parts) < 15 {
			continue
		}

		usdRate, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			continue
		}

		usdRateSell, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}

		goldType := parts[2]

		// For gold bars (positions 3,4 for buy/sell)
		buyBar, _ := strconv.ParseFloat(parts[3], 64)
		sellBar, _ := strconv.ParseFloat(parts[4], 64)

		// For ornaments/jewelry (positions 5,6 for buy/sell)
		buyOrnament, _ := strconv.ParseFloat(parts[5], 64)
		sellOrnament, _ := strconv.ParseFloat(parts[6], 64)

		prices = append(prices, GoldPrice{
			USDRate:      usdRate,
			USDRateSell:  usdRateSell,
			Type:         goldType,
			BuyBar:       buyBar,
			SellBar:      sellBar,
			BuyOrnament:  buyOrnament,
			SellOrnament: sellOrnament,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	return prices, nil
}

// LoadCombinedData loads and combines data from both files
func LoadCombinedData() (*CombinedData, error) {
	exchangeRates, err := ParseExchangeRates("raw-data/exrate.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to parse exchange rates: %v", err)
	}

	goldPrices, err := ParseGoldPrices("raw-data/market_retail.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to parse gold prices: %v", err)
	}

	return &CombinedData{
		Timestamp:     time.Now(),
		ExchangeRates: exchangeRates,
		GoldPrices:    goldPrices,
	}, nil
}