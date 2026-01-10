package websocket

import (
	"encoding/json"
	"fmt"
	"time"
)

// EventType represents the type of event (created, updated, deleted)
type EventType string

const (
	EventTypeCreated EventType = "created"
	EventTypeUpdated EventType = "updated"
	EventTypeDeleted EventType = "deleted"
	EventTypeBilled  EventType = "billed"
)

// EntityType represents the type of entity the event is about
type EntityType string

const (
	EntityTypeTransaction EntityType = "transaction"
)

// Event represents a WebSocket event message sent to clients
// Format: { type, entity, payload, timestamp }
type Event struct {
	Type      string      `json:"type"`      // Combined type e.g. "transaction.created"
	Entity    EntityType  `json:"entity"`    // Entity type e.g. "transaction"
	Payload   interface{} `json:"payload"`   // Full entity data
	Timestamp time.Time   `json:"timestamp"` // Event timestamp
}

// NewEvent creates a new event with the given type, entity, and payload
func NewEvent(eventType EventType, entityType EntityType, payload interface{}) Event {
	return Event{
		Type:      fmt.Sprintf("%s.%s", entityType, eventType),
		Entity:    entityType,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}
}

// ToJSON serializes the event to JSON bytes
func (e Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// TransactionCreated creates a transaction.created event
func TransactionCreated(payload interface{}) Event {
	return NewEvent(EventTypeCreated, EntityTypeTransaction, payload)
}

// TransactionUpdated creates a transaction.updated event
func TransactionUpdated(payload interface{}) Event {
	return NewEvent(EventTypeUpdated, EntityTypeTransaction, payload)
}

// TransactionDeleted creates a transaction.deleted event
func TransactionDeleted(payload interface{}) Event {
	return NewEvent(EventTypeDeleted, EntityTypeTransaction, payload)
}

// TransactionBilled creates a transaction.billed event
func TransactionBilled(payload interface{}) Event {
	return NewEvent(EventTypeBilled, EntityTypeTransaction, payload)
}
