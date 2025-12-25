package websocket

import (
	"context"
	"log"
	"net/http"

	"github.com/coder/websocket"
)

// Handler handles WebSocket connections
type Handler struct {
	hub *Hub
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// ServeHTTP implements http.Handler for WebSocket upgrade
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow all origins - configure properly in production
	})
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}

	client := NewClient(h.hub, conn)
	h.hub.Register(client)

	ctx := r.Context()

	// Start read and write pumps
	go client.WritePump(ctx)
	go client.ReadPump(ctx)
}

// HandleFunc returns an http.HandlerFunc for WebSocket connections
func (h *Handler) HandleFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}
}

// ServeWS is a standalone function to handle WebSocket connections
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}

	client := NewClient(hub, conn)
	hub.Register(client)

	ctx := context.Background()

	go client.WritePump(ctx)
	go client.ReadPump(ctx)
}
