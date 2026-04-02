package api

import (
	"net/http"

	"gold-socket/internal/config"
	"gold-socket/internal/websocket"
)

// Server represents the HTTP server
type Server struct {
	config   *config.ServerConfig
	hub      *websocket.Hub
	handlers *Handlers
	mux      *http.ServeMux
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.ServerConfig, hub *websocket.Hub) *Server {
	handlers := NewHandlers(cfg, hub)
	mux := http.NewServeMux()

	server := &Server{
		config:   cfg,
		hub:      hub,
		handlers: handlers,
		mux:      mux,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes() {
	// WebSocket endpoint
	wsHandler := websocket.NewHandler(s.hub, s.config.AllowedOrigins)
	s.mux.Handle("/ws", wsHandler)

	// API endpoints
	s.mux.HandleFunc("/health", s.handlers.HealthHandler)
	s.mux.HandleFunc("/api/data", s.handlers.DataHandler)
	s.mux.HandleFunc("/api/market-data", s.handlers.MarketDataHandler)
	s.mux.HandleFunc("/api/update-rate", s.handlers.UpdateRateHandler)

	// Static files
	s.mux.HandleFunc("/", StaticFileHandler(s.config.StaticDir))
}

// Handler returns the HTTP handler with middleware
func (s *Server) Handler() http.Handler {
	return LoggingMiddleware(s.mux)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := ":" + s.config.Port
	return http.ListenAndServe(addr, s.Handler())
}

// ListenAndServe starts the server on the specified address
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.Handler())
}
