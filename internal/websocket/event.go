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
	EntityTypeTransaction  EntityType = "transaction"
	EntityTypeRecurring    EntityType = "recurring"
	EntityTypeProjection   EntityType = "projection"
	EntityTypeSettlement   EntityType = "settlement"
	EntityTypeLoanPayment      EntityType = "loan_payment"
	EntityTypeLoanProvider     EntityType = "loan_provider"
	EntityTypeTransactionGroup EntityType = "transaction_group"
)

// Additional event types for specific events
const (
	EventTypeSynced          EventType = "synced"
	EventTypeBatchPaid       EventType = "batch_paid"
	EventTypeBatchUnpaid     EventType = "batch_unpaid"
	EventTypeChildrenChanged EventType = "children_changed"
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

// RecurringCreated creates a recurring.created event
func RecurringCreated(payload interface{}) Event {
	return NewEvent(EventTypeCreated, EntityTypeRecurring, payload)
}

// RecurringUpdated creates a recurring.updated event
func RecurringUpdated(payload interface{}) Event {
	return NewEvent(EventTypeUpdated, EntityTypeRecurring, payload)
}

// RecurringDeleted creates a recurring.deleted event
func RecurringDeleted(payload interface{}) Event {
	return NewEvent(EventTypeDeleted, EntityTypeRecurring, payload)
}

// ProjectionSynced creates a projection.synced event
func ProjectionSynced(payload interface{}) Event {
	return NewEvent(EventTypeSynced, EntityTypeProjection, payload)
}

// SettlementCreated creates a settlement.created event
func SettlementCreated(payload interface{}) Event {
	return NewEvent(EventTypeCreated, EntityTypeSettlement, payload)
}

// LoanPaymentBatchPaid creates a loan_payment.batch_paid event
func LoanPaymentBatchPaid(payload interface{}) Event {
	return NewEvent(EventTypeBatchPaid, EntityTypeLoanPayment, payload)
}

// LoanPaymentBatchUnpaid creates a loan_payment.batch_unpaid event
func LoanPaymentBatchUnpaid(payload interface{}) Event {
	return NewEvent(EventTypeBatchUnpaid, EntityTypeLoanPayment, payload)
}

// LoanProviderUpdated creates a loan_provider.updated event
func LoanProviderUpdated(payload interface{}) Event {
	return NewEvent(EventTypeUpdated, EntityTypeLoanProvider, payload)
}

// TransactionGroupCreated creates a transaction_group.created event
func TransactionGroupCreated(payload interface{}) Event {
	return NewEvent(EventTypeCreated, EntityTypeTransactionGroup, payload)
}

// TransactionGroupUpdated creates a transaction_group.updated event
func TransactionGroupUpdated(payload interface{}) Event {
	return NewEvent(EventTypeUpdated, EntityTypeTransactionGroup, payload)
}

// TransactionGroupChildrenChanged creates a transaction_group.children_changed event
func TransactionGroupChildrenChanged(payload interface{}) Event {
	return NewEvent(EventTypeChildrenChanged, EntityTypeTransactionGroup, payload)
}

// TransactionGroupDeleted creates a transaction_group.deleted event
func TransactionGroupDeleted(payload interface{}) Event {
	return NewEvent(EventTypeDeleted, EntityTypeTransactionGroup, payload)
}
