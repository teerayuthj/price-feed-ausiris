package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"gold-socket/pkg/models"
)

const (
	DefaultMarketRetailFile = "./raw-data/market_retail.txt"
	DefaultMarketJSONFile   = "./raw-data/market_data.json"
)

// ParseMarketData parses market_retail.txt and extracts required data
func ParseMarketData(filePath string) (*models.MarketData, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read market data file: %v", err)
	}

	lines := strings.Split(string(content), "\n")

	var spotUSD models.SpotData
	var g965bRetail models.GoldData
	var g9999kgRetail models.GoldData
	var g9999g models.GoldData
	found := map[string]bool{"spot": false, "g965b": false, "g9999kg": false, "g9999g": false}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 12 {
			continue
		}

		// Extract spot USD data (col 1, col 2) - first line has spot USD
		if !found["spot"] {
			bid, err1 := strconv.ParseFloat(parts[0], 64)
			offer, err2 := strconv.ParseFloat(parts[1], 64)

			if err1 == nil && err2 == nil && bid > 0 && offer > 0 {
				spotUSD.Bid = RoundToTwoDecimals(bid)
				spotUSD.Offer = RoundToTwoDecimals(offer)
				found["spot"] = true
			}
		}

		// Extract G965B (Retail) data
		if strings.Contains(parts[2], "G965B (Retail)") {
			if len(parts) >= 12 {
				bid, err1 := strconv.ParseFloat(parts[9], 64)
				offer, err2 := strconv.ParseFloat(parts[10], 64)

				if err1 == nil && err2 == nil && bid > 0 && offer > 0 {
					g965bRetail.Bid = RoundToTwoDecimals(bid)
					g965bRetail.Offer = RoundToTwoDecimals(offer)
					found["g965b"] = true
				}
			}
		}

		// Extract G9999KG (Retail) data - multiply by 65.6 for KG conversion
		if strings.Contains(parts[2], "G9999KG (Retail)") {
			if len(parts) >= 12 {
				bid, err1 := strconv.ParseFloat(parts[9], 64)
				offer, err2 := strconv.ParseFloat(parts[10], 64)

				if err1 == nil && err2 == nil && bid > 0 && offer > 0 {
					g9999kgRetail.Bid = RoundToTwoDecimals(bid * 65.6)
					g9999kgRetail.Offer = RoundToTwoDecimals(offer * 65.6)
					found["g9999kg"] = true
				}
			}
		}

		// Extract G9999G data
		if strings.Contains(parts[2], "G9999G") {
			if len(parts) >= 12 {
				bid, err1 := strconv.ParseFloat(parts[9], 64)
				offer, err2 := strconv.ParseFloat(parts[10], 64)

				if err1 == nil && err2 == nil && bid > 0 && offer > 0 {
					g9999g.Bid = RoundToTwoDecimals(bid)
					g9999g.Offer = RoundToTwoDecimals(offer)
					found["g9999g"] = true
				}
			}
		}

		if found["spot"] && found["g965b"] && found["g9999kg"] && found["g9999g"] {
			break
		}
	}

	if !found["spot"] || !found["g965b"] || !found["g9999kg"] || !found["g9999g"] {
		return nil, fmt.Errorf("failed to extract required data - spot: %v, g965b: %v, g9999kg: %v, g9999g: %v",
			found["spot"], found["g965b"], found["g9999kg"], found["g9999g"])
	}

	now := NowInBangkok()

	marketData := &models.MarketData{
		Timestamp:     now,
		Time:          now.Format("15:04:05"),
		SpotUSD:       spotUSD,
		G965BRetail:   g965bRetail,
		G9999KGRetail: g9999kgRetail,
		G9999G:        g9999g,
		MarketStatus:  GetMarketStatus(),
		Source:        SourceAuto,
	}

	return marketData, nil
}

// SaveMarketDataJSON saves market data to JSON file
func SaveMarketDataJSON(data *models.MarketData, filePath string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal market data: %v", err)
	}

	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write market data JSON: %v", err)
	}

	log.Printf("Market data saved to %s", filePath)
	return nil
}

// ProcessMarketRetailData processes market_retail.txt and creates JSON
func ProcessMarketRetailData() error {
	if _, err := os.Stat(DefaultMarketRetailFile); os.IsNotExist(err) {
		return fmt.Errorf("market retail file not found: %s", DefaultMarketRetailFile)
	}

	marketData, err := ParseMarketData(DefaultMarketRetailFile)
	if err != nil {
		return fmt.Errorf("failed to parse market data: %v", err)
	}

	err = SaveMarketDataJSON(marketData, DefaultMarketJSONFile)
	if err != nil {
		return fmt.Errorf("failed to save market data JSON: %v", err)
	}

	log.Printf("Market data processed successfully - Spot USD: %.2f/%.2f, G965B: %.0f/%.0f, G9999KG: %.2f/%.2f, G9999G: %.2f/%.2f",
		marketData.SpotUSD.Bid, marketData.SpotUSD.Offer,
		marketData.G965BRetail.Bid, marketData.G965BRetail.Offer,
		marketData.G9999KGRetail.Bid, marketData.G9999KGRetail.Offer,
		marketData.G9999G.Bid, marketData.G9999G.Offer)

	return nil
}

// LoadMarketDataFromJSON loads market data from JSON file
func LoadMarketDataFromJSON() (*models.MarketData, error) {
	if _, err := os.Stat(DefaultMarketJSONFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("market data JSON file does not exist")
	}

	data, err := os.ReadFile(DefaultMarketJSONFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read market data JSON: %v", err)
	}

	var marketData models.MarketData
	err = json.Unmarshal(data, &marketData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal market data: %v", err)
	}

	return &marketData, nil
}

// ValidateMarketData validates market data before processing
func ValidateMarketData(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Failed to read market data file for validation: %v", err)
		return false
	}

	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 12 {
			continue
		}

		bid, err1 := strconv.ParseFloat(parts[0], 64)
		offer, err2 := strconv.ParseFloat(parts[1], 64)

		if err1 == nil && err2 == nil && bid > 0 && offer > 0 {
			return true
		}
	}

	log.Printf("Market data validation failed - no valid spot USD data found")
	return false
}
