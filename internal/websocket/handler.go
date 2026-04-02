package websocket

import (
	"context"
	"log"
	"net/http"

	"github.com/coder/websocket"
)

// Handler handles WebSocket connections
type Handler struct {
	hub            *Hub
	originPatterns []string
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, originPatterns []string) *Handler {
	return &Handler{
		hub:            hub,
		originPatterns: originPatterns,
	}
}

// ServeHTTP implements http.Handler for WebSocket upgrade
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.originPatterns,
	})
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}

	client := NewClient(h.hub, conn)
	h.hub.Register(client)

	// Use background context - r.Context() gets canceled when handler returns
	ctx, cancel := context.WithCancel(context.Background())

	// Start read and write pumps
	go func() {
		defer cancel()
		client.WritePump(ctx)
	}()
	go func() {
		defer cancel()
		client.ReadPump(ctx)
	}()
}

// HandleFunc returns an http.HandlerFunc for WebSocket connections
func (h *Handler) HandleFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}
}

// ServeWS is a standalone function to handle WebSocket connections
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}

	client := NewClient(hub, conn)
	hub.Register(client)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		client.WritePump(ctx)
	}()
	go func() {
		defer cancel()
		client.ReadPump(ctx)
	}()
}
