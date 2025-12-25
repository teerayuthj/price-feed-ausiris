package models

import "time"

// MarketData represents the processed market data structure
type MarketData struct {
	Timestamp     time.Time `json:"timestamp"`
	Time          string    `json:"time"`
	SpotUSD       SpotData  `json:"spot_usd"`
	G965BRetail   GoldData  `json:"g965b_retail"`
	G9999KGRetail GoldData  `json:"g9999kg_retail"`
	G9999G        GoldData  `json:"g9999g"`
	MarketStatus  string    `json:"market_status"`
	Source        string    `json:"source"`
}

// SpotData represents spot USD data
type SpotData struct {
	Bid   float64 `json:"bid"`
	Offer float64 `json:"offer"`
}

// GoldData represents gold pricing data
type GoldData struct {
	Bid   float64 `json:"bid"`
	Offer float64 `json:"offer"`
}

// WebSocketMessage wraps data for WebSocket transmission
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}
