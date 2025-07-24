package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin
	},
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

// Client represents a WebSocket client
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("Client connected. Total clients: %d", len(h.clients))
			
			// Send current USD rate to new client
			go func() {
				data, err := GetCurrentUSDRate()
				if err != nil {
					log.Printf("Error loading USD rate for new client: %v", err)
					return
				}
				
				jsonData, err := json.Marshal(data)
				if err != nil {
					log.Printf("Error marshaling USD rate for new client: %v", err)
					return
				}
				
				select {
				case client.send <- jsonData:
				default:
					close(client.send)
					h.mutex.Lock()
					delete(h.clients, client)
					h.mutex.Unlock()
				}
			}()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client disconnected. Total clients: %d", len(h.clients))
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastData broadcasts latest USD rate to all clients
func (h *Hub) BroadcastData() {
	data, err := GetCurrentUSDRate()
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
	log.Printf("Broadcasted USD rate to %d clients: Buy=%.2f, Sell=%.2f [%s/%s]", 
		len(h.clients), data.Buy, data.Sell, data.MarketStatus, data.Source)
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// StartFileWatcher starts watching for file changes
func StartFileWatcher(hub *Hub) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Printf("File modified: %s", event.Name)
					// Broadcast updated data
					hub.BroadcastData()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("File watcher error: %v", err)
			}
		}
	}()

	// Watch the raw-data directory
	err = watcher.Add("raw-data")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("File watcher started, monitoring raw-data directory")
	select {} // Block forever
}

// WebSocket handler
func wsHandler(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

// HTTP handler for serving static files
func staticFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		r.URL.Path = "/index.html"
	}
	
	filePath := filepath.Join("static", r.URL.Path[1:])
	http.ServeFile(w, r, filePath)
}

// API handler to get current USD rate as JSON
func apiHandler(w http.ResponseWriter, r *http.Request) {
	data, err := GetCurrentUSDRate()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading USD rate: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(data)
}

// API handler to update USD rate manually
func updateRateHandler(w http.ResponseWriter, r *http.Request) {
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

	err = CreateManualUSDRate(request.Buy, request.Sell)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating USD rate: %v", err), http.StatusInternalServerError)
		return
	}

	// Broadcast the update
	// You'll need to pass the hub to this handler or make it global
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "USD rate updated"})
}

// StartWebSocketServer starts the WebSocket server with scheduled downloads
func StartWebSocketServer() {
	hub := NewHub()
	go hub.Run()
	
	// Start file watcher in a separate goroutine
	go StartFileWatcher(hub)
	
	// Get configuration for scheduler
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}
	
	// Get download interval from environment (default: 30 seconds)
	intervalSeconds := 30
	if envInterval := os.Getenv("DOWNLOAD_INTERVAL_SECONDS"); envInterval != "" {
		if parsed, err := strconv.Atoi(envInterval); err == nil {
			intervalSeconds = parsed
		}
	}
	
	// Get WebSocket port from environment (default: 8080)
	port := "8080"
	if envPort := os.Getenv("WEBSOCKET_PORT"); envPort != "" {
		port = envPort
	}
	
	// Start scheduled downloads
	downloadInterval := time.Duration(intervalSeconds) * time.Second
	go StartScheduledDownloads(hub, downloadInterval)

	// Set up HTTP routes
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler(hub, w, r)
	})
	http.HandleFunc("/api/data", apiHandler)
	http.HandleFunc("/api/update-rate", func(w http.ResponseWriter, r *http.Request) {
		updateRateHandler(w, r)
		// Broadcast after manual update
		if r.Method == http.MethodPost {
			go hub.BroadcastData()
		}
	})
	http.HandleFunc("/", staticFileHandler)

	log.Printf("WebSocket server starting on :%s", port)
	log.Printf("WebSocket endpoint: ws://localhost:%s/ws", port)
	log.Printf("API endpoint: http://localhost:%s/api/data", port)
	log.Printf("Web interface: http://localhost:%s", port)
	log.Printf("Scheduled downloads every %d seconds", intervalSeconds)
	
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}