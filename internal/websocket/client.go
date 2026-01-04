package websocket

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	// writeWait is time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// pongWait is time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// pingPeriod is the interval for sending pings (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize is maximum message size allowed from peer
	maxMessageSize = 512
)

// Client represents a single WebSocket connection
type Client struct {
	id          string
	workspaceID int32
	conn        *websocket.Conn
	hub         *Hub
	send        chan []byte
	closed      bool
	mu          sync.RWMutex
	closeOnce   sync.Once
}

// NewClient creates a new WebSocket client
func NewClient(conn *websocket.Conn, workspaceID int32, hub *Hub) *Client {
	return &Client{
		id:          uuid.New().String(),
		workspaceID: workspaceID,
		conn:        conn,
		hub:         hub,
		send:        make(chan []byte, 256),
	}
}

// ID returns the client's unique identifier
func (c *Client) ID() string {
	return c.id
}

// WorkspaceID returns the client's workspace ID
func (c *Client) WorkspaceID() int32 {
	return c.workspaceID
}

// Send queues a message to be sent to the client
func (c *Client) Send(data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	select {
	case c.send <- data:
		return nil
	default:
		// Buffer is full, client is too slow
		return ErrClientClosed
	}
}

// Close closes the client connection
// Safe to call multiple times from different goroutines
func (c *Client) Close() error {
	var closeErr error
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		close(c.send)
		c.mu.Unlock()

		closeErr = c.conn.Close()
	})
	return closeErr
}

// IsClosed returns whether the client is closed
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// ReadPump pumps messages from the WebSocket connection
// This should be run in a goroutine
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Warn().
					Err(err).
					Str("client_id", c.id).
					Int32("workspace_id", c.workspaceID).
					Msg("WebSocket unexpected close")
			}
			break
		}
		// We don't expect clients to send messages in this story
		// Future stories may add client->server messaging
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
// This should be run in a goroutine
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed, hub closed this client
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Warn().
					Err(err).
					Str("client_id", c.id).
					Int32("workspace_id", c.workspaceID).
					Msg("WebSocket write error")
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
