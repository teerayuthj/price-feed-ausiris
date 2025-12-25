package websocket

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"

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
}

// NewHub creates a new Hub
func NewHub(redisClient *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		redis:      redisClient,
	}
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

// sendInitialData sends current data to a newly connected client
func (h *Hub) sendInitialData(client *Client) {
	// Recover from panic if channel is closed during send
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in sendInitialData: %v", r)
		}
	}()

	data, err := parser.GetCurrentUSDRate()
	if err != nil {
		log.Printf("Error loading USD rate for new client: %v", err)
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling USD rate for new client: %v", err)
		return
	}

	// Hold lock while checking and sending to prevent race condition
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if _, stillConnected := h.clients[client]; !stillConnected {
		return
	}

	// Try to send with non-blocking select
	select {
	case client.send <- jsonData:
		// Successfully sent
	default:
		// Channel full or closed, skip
	}
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

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling USD rate: %v", err)
		return
	}

	h.broadcast <- jsonData

	// Publish to Redis for other instances
	if h.redis != nil {
		ctx := context.Background()
		h.redis.Publish(ctx, redis.ChannelUSDRate, jsonData)
	}

	log.Printf("Broadcasted USD rate to %d clients: Buy=%.2f, Sell=%.2f [%s/%s]",
		len(h.clients), data.Buy, data.Sell, data.MarketStatus, data.Source)

	// Also broadcast market data
	h.BroadcastMarketData()
}

// BroadcastMarketData broadcasts latest market data to all clients
func (h *Hub) BroadcastMarketData() {
	filePath := "./raw-data/market_data.json"

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("Market data file not found: %s", filePath)
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading market data: %v", err)
		return
	}

	message := models.WebSocketMessage{
		Type: "market_data",
		Data: json.RawMessage(data),
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling market data message: %v", err)
		return
	}

	h.broadcast <- jsonMessage

	// Publish to Redis for other instances
	if h.redis != nil {
		ctx := context.Background()
		h.redis.Publish(ctx, redis.ChannelMarketData, jsonMessage)
	}

	log.Printf("Broadcasted market data to %d clients", len(h.clients))
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
