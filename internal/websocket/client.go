package websocket

import (
	"context"
	"log"
	"time"

	"github.com/coder/websocket"
)

const (
	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = 54 * time.Second

	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// Client represents a WebSocket client
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump(ctx context.Context) {
	defer func() {
		c.hub.Unregister(c)
		c.conn.CloseNow()
	}()

	c.conn.SetReadLimit(maxMessageSize)

	for {
		readCtx, cancel := context.WithTimeout(ctx, pongWait)
		_, _, err := c.conn.Read(readCtx)
		cancel()

		if err != nil {
			status := websocket.CloseStatus(err)
			if status != websocket.StatusNormalClosure && status != websocket.StatusGoingAway {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump(ctx context.Context) {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		c.conn.CloseNow()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.Close(websocket.StatusGoingAway, "channel closed")
				return
			}

			writeCtx, cancel := context.WithTimeout(ctx, writeWait)
			err := c.conn.Write(writeCtx, websocket.MessageText, message)
			cancel()

			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

			// Drain queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				msg := <-c.send
				writeCtx, cancel := context.WithTimeout(ctx, writeWait)
				err := c.conn.Write(writeCtx, websocket.MessageText, msg)
				cancel()
				if err != nil {
					log.Printf("WebSocket write error: %v", err)
					return
				}
			}

		case <-pingTicker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeWait)
			err := c.conn.Ping(pingCtx)
			cancel()

			if err != nil {
				log.Printf("WebSocket ping error: %v", err)
				return
			}

		case <-ctx.Done():
			c.conn.Close(websocket.StatusGoingAway, "context cancelled")
			return
		}
	}
}

// Send sends a message to the client
func (c *Client) Send(data []byte) bool {
	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}
