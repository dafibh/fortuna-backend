package websocket

// EventPublisher defines the interface for publishing events to WebSocket clients
type EventPublisher interface {
	// Publish sends an event to all clients connected to the specified workspace
	Publish(workspaceID int32, event Event)
}

// Ensure Hub implements EventPublisher
var _ EventPublisher = (*Hub)(nil)

// Publish implements EventPublisher by broadcasting the event to the workspace
func (h *Hub) Publish(workspaceID int32, event Event) {
	h.Broadcast(workspaceID, event)
}

// NoOpPublisher is a publisher that does nothing (for testing or when WebSocket is disabled)
type NoOpPublisher struct{}

// Publish does nothing
func (n *NoOpPublisher) Publish(workspaceID int32, event Event) {}
