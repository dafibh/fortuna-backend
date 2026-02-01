package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventType_String(t *testing.T) {
	tests := []struct {
		name     string
		et       EventType
		expected string
	}{
		{"created", EventTypeCreated, "created"},
		{"updated", EventTypeUpdated, "updated"},
		{"deleted", EventTypeDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.et))
		})
	}
}

func TestEntityType_String(t *testing.T) {
	tests := []struct {
		name     string
		et       EntityType
		expected string
	}{
		{"transaction", EntityTypeTransaction, "transaction"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.et))
		})
	}
}

func TestNewEvent(t *testing.T) {
	payload := map[string]interface{}{
		"id":     1,
		"name":   "Test Transaction",
		"amount": "100.00",
	}

	before := time.Now()
	evt := NewEvent(EventTypeCreated, EntityTypeTransaction, payload)
	after := time.Now()

	assert.Equal(t, "transaction.created", evt.Type)
	assert.Equal(t, EntityTypeTransaction, evt.Entity)
	assert.Equal(t, payload, evt.Payload)
	assert.True(t, !evt.Timestamp.Before(before) && !evt.Timestamp.After(after))
}

func TestEvent_JSON_Serialization(t *testing.T) {
	fixedTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	payload := map[string]interface{}{
		"id":     float64(1),
		"name":   "Test Transaction",
		"amount": "100.00",
	}

	evt := Event{
		Type:      "transaction.created",
		Entity:    EntityTypeTransaction,
		Payload:   payload,
		Timestamp: fixedTime,
	}

	data, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, evt.Type, decoded.Type)
	assert.Equal(t, evt.Entity, decoded.Entity)
	assert.Equal(t, fixedTime.UTC(), decoded.Timestamp.UTC())

	// Payload should be preserved
	decodedPayload, ok := decoded.Payload.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), decodedPayload["id"])
	assert.Equal(t, "Test Transaction", decodedPayload["name"])
	assert.Equal(t, "100.00", decodedPayload["amount"])
}

func TestEvent_ToJSON(t *testing.T) {
	payload := map[string]interface{}{
		"id": float64(42),
	}

	evt := NewEvent(EventTypeUpdated, EntityTypeTransaction, payload)

	data, err := evt.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify it's valid JSON
	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "transaction.updated", decoded["type"])
	assert.Equal(t, "transaction", decoded["entity"])
	assert.NotNil(t, decoded["payload"])
	assert.NotNil(t, decoded["timestamp"])
}

func TestTransactionGroupEvent_Helpers(t *testing.T) {
	payload := map[string]interface{}{
		"id":         float64(1),
		"name":       "Groceries",
		"month":      "2026-01",
		"childCount": float64(3),
	}

	t.Run("TransactionGroupCreated", func(t *testing.T) {
		evt := TransactionGroupCreated(payload)
		assert.Equal(t, "transaction_group.created", evt.Type)
		assert.Equal(t, EntityTypeTransactionGroup, evt.Entity)
		assert.Equal(t, payload, evt.Payload)
	})

	t.Run("TransactionGroupUpdated", func(t *testing.T) {
		evt := TransactionGroupUpdated(payload)
		assert.Equal(t, "transaction_group.updated", evt.Type)
		assert.Equal(t, EntityTypeTransactionGroup, evt.Entity)
		assert.Equal(t, payload, evt.Payload)
	})

	t.Run("TransactionGroupDeleted", func(t *testing.T) {
		evt := TransactionGroupDeleted(payload)
		assert.Equal(t, "transaction_group.deleted", evt.Type)
		assert.Equal(t, EntityTypeTransactionGroup, evt.Entity)
		assert.Equal(t, payload, evt.Payload)
	})

	t.Run("TransactionGroupChildrenChanged", func(t *testing.T) {
		evt := TransactionGroupChildrenChanged(payload)
		assert.Equal(t, "transaction_group.children_changed", evt.Type)
		assert.Equal(t, EntityTypeTransactionGroup, evt.Entity)
		assert.Equal(t, payload, evt.Payload)
	})
}

func TestEntityTypeTransactionGroup_String(t *testing.T) {
	assert.Equal(t, "transaction_group", string(EntityTypeTransactionGroup))
}

func TestEventTypeChildrenChanged_String(t *testing.T) {
	assert.Equal(t, "children_changed", string(EventTypeChildrenChanged))
}

func TestTransactionEvent_Helpers(t *testing.T) {
	txPayload := map[string]interface{}{
		"id":     float64(1),
		"name":   "Grocery shopping",
		"amount": "50.00",
	}

	t.Run("TransactionCreated", func(t *testing.T) {
		evt := TransactionCreated(txPayload)
		assert.Equal(t, "transaction.created", evt.Type)
		assert.Equal(t, EntityTypeTransaction, evt.Entity)
		assert.Equal(t, txPayload, evt.Payload)
	})

	t.Run("TransactionUpdated", func(t *testing.T) {
		evt := TransactionUpdated(txPayload)
		assert.Equal(t, "transaction.updated", evt.Type)
		assert.Equal(t, EntityTypeTransaction, evt.Entity)
		assert.Equal(t, txPayload, evt.Payload)
	})

	t.Run("TransactionDeleted", func(t *testing.T) {
		evt := TransactionDeleted(txPayload)
		assert.Equal(t, "transaction.deleted", evt.Type)
		assert.Equal(t, EntityTypeTransaction, evt.Entity)
		assert.Equal(t, txPayload, evt.Payload)
	})
}
