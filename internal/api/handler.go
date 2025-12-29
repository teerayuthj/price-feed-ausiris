package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"gold-socket/internal/parser"
	"gold-socket/internal/websocket"
)

// Handlers holds HTTP handler dependencies
type Handlers struct {
	hub *websocket.Hub
}

// NewHandlers creates new HTTP handlers
func NewHandlers(hub *websocket.Hub) *Handlers {
	return &Handlers{hub: hub}
}

// HealthHandler handles health check requests
func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// DataHandler handles USD rate API requests
func (h *Handlers) DataHandler(w http.ResponseWriter, r *http.Request) {
	data, err := parser.GetCurrentUSDRate()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading USD rate: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(data)
}

// MarketDataHandler handles market data API requests
func (h *Handlers) MarketDataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if h.hub != nil {
		payload, err := h.hub.PrepareMarketDataPayload()
		if err == nil {
			w.Write(payload)
			return
		}
		if os.IsNotExist(err) {
			http.Error(w, "Market data not available", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Error reading market data: %v", err), http.StatusInternalServerError)
		return
	}

	filePath := "./raw-data/market_data.json"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Market data not available", http.StatusNotFound)
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading market data: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

// UpdateRateHandler handles manual rate update requests
func (h *Handlers) UpdateRateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Buy  float64 `json:"buy"`
		Sell float64 `json:"sell"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Buy <= 0 || request.Sell <= 0 {
		http.Error(w, "Buy and sell rates must be positive", http.StatusBadRequest)
		return
	}

	err = parser.CreateManualUSDRate(request.Buy, request.Sell)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating USD rate: %v", err), http.StatusInternalServerError)
		return
	}

	// Broadcast the update
	if h.hub != nil {
		go h.hub.BroadcastData()
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "USD rate updated",
	})
}

// StaticFileHandler serves static files
func StaticFileHandler(staticDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		filePath := staticDir + path

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// Try serving index.html for SPA routing
			filePath = staticDir + "/index.html"
		}

		http.ServeFile(w, r, filePath)
	}
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
