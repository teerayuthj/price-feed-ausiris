package parser

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultDataDir = "./raw-data"

func resolveDataDir() string {
	value := strings.TrimSpace(os.Getenv("DATA_DIR"))
	if value == "" {
		return defaultDataDir
	}
	return value
}

func MarketRetailFilePath() string {
	return filepath.Join(resolveDataDir(), "market_retail.txt")
}

func MarketJSONFilePath() string {
	return filepath.Join(resolveDataDir(), "market_data.json")
}

func ExrateFilePath() string {
	return filepath.Join(resolveDataDir(), "exrate.txt")
}

func USDJSONFilePath() string {
	return filepath.Join(resolveDataDir(), "usd_rate.json")
}
