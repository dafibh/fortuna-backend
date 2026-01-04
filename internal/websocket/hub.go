package websocket

import (
	"errors"
	"sync"

	"github.com/rs/zerolog/log"
)

// ErrClientClosed is returned when attempting to send to a closed client
var ErrClientClosed = errors.New("client is closed")

// ClientInterface defines the interface that clients must implement
type ClientInterface interface {
	ID() string
	WorkspaceID() int32
	Send(data []byte) error
	Close() error
}

// Hub manages WebSocket connections organized by workspace
// It is safe for concurrent use
type Hub struct {
	// workspaces maps workspace ID to a map of client ID to client
	workspaces map[int32]map[string]ClientInterface
	mu         sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		workspaces: make(map[int32]map[string]ClientInterface),
	}
}

// Register adds a client to the hub under its workspace
func (h *Hub) Register(client ClientInterface) {
	h.mu.Lock()
	defer h.mu.Unlock()

	workspaceID := client.WorkspaceID()
	clientID := client.ID()

	if h.workspaces[workspaceID] == nil {
		h.workspaces[workspaceID] = make(map[string]ClientInterface)
	}

	h.workspaces[workspaceID][clientID] = client

	log.Debug().
		Int32("workspace_id", workspaceID).
		Str("client_id", clientID).
		Msg("WebSocket client registered")
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client ClientInterface) {
	h.mu.Lock()
	defer h.mu.Unlock()

	workspaceID := client.WorkspaceID()
	clientID := client.ID()

	if clients, ok := h.workspaces[workspaceID]; ok {
		if _, exists := clients[clientID]; exists {
			delete(clients, clientID)

			// Clean up empty workspace maps
			if len(clients) == 0 {
				delete(h.workspaces, workspaceID)
			}

			log.Debug().
				Int32("workspace_id", workspaceID).
				Str("client_id", clientID).
				Msg("WebSocket client unregistered")
		}
	}
}

// Broadcast sends an event to all clients in a specific workspace
func (h *Hub) Broadcast(workspaceID int32, event Event) {
	data, err := event.ToJSON()
	if err != nil {
		log.Error().
			Err(err).
			Int32("workspace_id", workspaceID).
			Str("event_type", event.Type).
			Msg("Failed to serialize event")
		return
	}

	h.mu.RLock()
	clients, ok := h.workspaces[workspaceID]
	if !ok || len(clients) == 0 {
		h.mu.RUnlock()
		return
	}

	// Copy clients to avoid holding lock during send
	clientsCopy := make([]ClientInterface, 0, len(clients))
	for _, client := range clients {
		clientsCopy = append(clientsCopy, client)
	}
	h.mu.RUnlock()

	// Send to each client asynchronously
	for _, client := range clientsCopy {
		go func(c ClientInterface) {
			if err := c.Send(data); err != nil {
				log.Warn().
					Err(err).
					Int32("workspace_id", workspaceID).
					Str("client_id", c.ID()).
					Msg("Failed to send to client")
			}
		}(client)
	}

	log.Debug().
		Int32("workspace_id", workspaceID).
		Str("event_type", event.Type).
		Int("client_count", len(clientsCopy)).
		Msg("Broadcast event")
}

// ClientCount returns the number of clients connected to a workspace
func (h *Hub) ClientCount(workspaceID int32) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.workspaces[workspaceID]; ok {
		return len(clients)
	}
	return 0
}

// TotalClientCount returns the total number of connected clients across all workspaces
func (h *Hub) TotalClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, clients := range h.workspaces {
		total += len(clients)
	}
	return total
}
