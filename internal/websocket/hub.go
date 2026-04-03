package websocket

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"gold-socket/internal/parser"
	"gold-socket/internal/redis"
	"gold-socket/pkg/models"
)

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
	redis      *redis.Client

	// Source status tracking
	sourceStatus     string // "connected" or "disconnected"
	lastSuccessTime  string // timestamp of last successful SFTP update
	lastError        string // last error message
	sourceMutex      sync.RWMutex
}

// NewHub creates a new Hub
func NewHub(redisClient *redis.Client) *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		broadcast:    make(chan []byte, 256),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		redis:        redisClient,
		sourceStatus: "disconnected", // Start as disconnected until first successful fetch
	}
}

// SetSourceConnected marks the data source as connected
func (h *Hub) SetSourceConnected() {
	h.sourceMutex.Lock()
	defer h.sourceMutex.Unlock()
	h.sourceStatus = "connected"
	h.lastSuccessTime = parser.NowInBangkok().Format(time.RFC3339)
	h.lastError = ""
}

// SetSourceDisconnected marks the data source as disconnected with error
func (h *Hub) SetSourceDisconnected(err string) {
	h.sourceMutex.Lock()
	defer h.sourceMutex.Unlock()
	h.sourceStatus = "disconnected"
	h.lastError = err
}

// GetSourceStatus returns current source status
func (h *Hub) GetSourceStatus() (status string, lastSuccess string, lastErr string) {
	h.sourceMutex.RLock()
	defer h.sourceMutex.RUnlock()
	return h.sourceStatus, h.lastSuccessTime, h.lastError
}

// Run starts the hub event loop
func (h *Hub) Run(ctx context.Context) {
	// Subscribe to Redis channels for multi-instance support
	if h.redis != nil {
		go h.subscribeToRedis(ctx)
	}

	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("Client connected. Total clients: %d", len(h.clients))

			// Send current USD rate to new client
			go h.sendInitialData(client)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client disconnected. Total clients: %d", len(h.clients))
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.broadcastToClients(message)

		case <-ctx.Done():
			log.Println("Hub shutting down")
			return
		}
	}
}

// sendInitialData sends current market data to a newly connected client
func (h *Hub) sendInitialData(client *Client) {
	// Recover from panic if channel is closed during send
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in sendInitialData: %v", r)
		}
	}()

	// Hold lock while checking and sending to prevent race condition
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if _, stillConnected := h.clients[client]; !stillConnected {
		return
	}

	// Load and send market data with fresh market_status
	jsonMessage, err := h.prepareMarketDataMessage()
	if err != nil {
		log.Printf("Error preparing market data for new client: %v", err)
		return
	}

	select {
	case client.send <- jsonMessage:
		log.Printf("Sent initial market data to client")
	default:
		// Channel full or closed, skip
	}
}

// PrepareMarketDataPayload loads market data and injects runtime status fields.
func (h *Hub) PrepareMarketDataPayload() ([]byte, error) {
	filePath := parser.MarketJSONFilePath()

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var marketData map[string]interface{}
	if err := json.Unmarshal(data, &marketData); err != nil {
		return nil, err
	}

	// Get source status
	sourceStatus, lastSuccess, lastErr := h.GetSourceStatus()
	sourceConnected := sourceStatus == "connected"

	// Check if price is valid (g965b_retail bid/offer must be > 0)
	priceValid := h.isPriceValid(marketData)

	// Update market_status based on time, source connection, and price validity
	marketData["market_status"] = parser.GetMarketStatusWithData(sourceConnected, priceValid)

	// Add source status info
	marketData["source_status"] = sourceStatus
	if lastSuccess != "" {
		marketData["last_success_time"] = lastSuccess
	}
	if lastErr != "" {
		marketData["last_error"] = lastErr
	}

	return json.Marshal(marketData)
}

// isPriceValid checks if the market data has valid prices (not zero)
func (h *Hub) isPriceValid(marketData map[string]interface{}) bool {
	// Check g965b_retail prices
	if g965b, ok := marketData["g965b_retail"].(map[string]interface{}); ok {
		bid, bidOk := g965b["bid"].(float64)
		offer, offerOk := g965b["offer"].(float64)
		if bidOk && offerOk && bid > 0 && offer > 0 {
			return true
		}
	}
	return false
}

// prepareMarketDataMessage loads market data and updates market_status in real-time
func (h *Hub) prepareMarketDataMessage() ([]byte, error) {
	updatedData, err := h.PrepareMarketDataPayload()
	if err != nil {
		return nil, err
	}

	message := models.WebSocketMessage{
		Type: "market_data",
		Data: json.RawMessage(updatedData),
	}

	return json.Marshal(message)
}

// broadcastToClients sends message to all connected clients
func (h *Hub) broadcastToClients(message []byte) {
	h.mutex.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mutex.RUnlock()

	for _, client := range clients {
		select {
		case client.send <- message:
			// Successfully sent
		default:
			// Channel full or client slow, trigger unregister
			go h.Unregister(client)
		}
	}
}

// BroadcastData broadcasts latest USD rate to all clients
func (h *Hub) BroadcastData() {
	data, err := parser.GetCurrentUSDRate()
	if err != nil {
		log.Printf("Error loading USD rate: %v", err)
		return
	}

	// Get source status and check price validity
	sourceStatus, _, _ := h.GetSourceStatus()
	sourceConnected := sourceStatus == "connected"
	priceValid := data.Buy > 0 && data.Sell > 0

	// Override MarketStatus based on source connection and price validity
	data.MarketStatus = parser.GetMarketStatusWithData(sourceConnected, priceValid)

	// Wrap USD rate data in WebSocketMessage format for consistency
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling USD rate: %v", err)
		return
	}

	message := models.WebSocketMessage{
		Type: "usd_rate",
		Data: json.RawMessage(jsonData),
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling USD rate message: %v", err)
		return
	}

	h.broadcast <- jsonMessage

	// Publish to Redis for other instances
	if h.redis != nil {
		ctx := context.Background()
		h.redis.Publish(ctx, redis.ChannelUSDRate, jsonMessage)
	}

	log.Printf("Broadcasted USD rate to %d clients: Buy=%.2f, Sell=%.2f [%s/%s]",
		len(h.clients), data.Buy, data.Sell, data.MarketStatus, data.Source)

	// Also broadcast market data
	h.BroadcastMarketData()
}

// BroadcastMarketData broadcasts latest market data to all clients
func (h *Hub) BroadcastMarketData() {
	jsonMessage, err := h.prepareMarketDataMessage()
	if err != nil {
		log.Printf("Error preparing market data for broadcast: %v", err)
		return
	}

	h.broadcast <- jsonMessage

	// Publish to Redis for other instances
	if h.redis != nil {
		ctx := context.Background()
		h.redis.Publish(ctx, redis.ChannelMarketData, jsonMessage)
	}

	log.Printf("Broadcasted market data to %d clients (market_status: %s)", len(h.clients), parser.GetMarketStatus())
}

// subscribeToRedis subscribes to Redis channels for multi-instance support
func (h *Hub) subscribeToRedis(ctx context.Context) {
	// Subscribe to USD rate channel
	go func() {
		err := h.redis.Subscribe(ctx, redis.ChannelUSDRate, func(data []byte) {
			h.broadcastToClients(data)
		})
		if err != nil {
			log.Printf("Redis USD rate subscription error: %v", err)
		}
	}()

	// Subscribe to market data channel
	go func() {
		err := h.redis.Subscribe(ctx, redis.ChannelMarketData, func(data []byte) {
			h.broadcastToClients(data)
		})
		if err != nil {
			log.Printf("Redis market data subscription error: %v", err)
		}
	}()
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// Register registers a client with the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}
