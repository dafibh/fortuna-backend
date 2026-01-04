package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHub_Implements_EventPublisher(t *testing.T) {
	// Compile-time check that Hub implements EventPublisher
	var _ EventPublisher = (*Hub)(nil)
}

func TestHub_Publish(t *testing.T) {
	hub := NewHub()

	// Create mock client
	client := newMockClient("client-1", 1)
	hub.Register(client)

	// Publish event via EventPublisher interface
	var publisher EventPublisher = hub
	event := TransactionCreated(map[string]interface{}{"id": float64(42)})
	publisher.Publish(1, event)

	// Allow async broadcast to complete
	time.Sleep(10 * time.Millisecond)

	// Verify client received the event
	messages := client.GetMessages()
	assert.Len(t, messages, 1)
}

func TestNoOpPublisher_Publish(t *testing.T) {
	publisher := &NoOpPublisher{}

	// Should not panic
	assert.NotPanics(t, func() {
		event := TransactionCreated(map[string]interface{}{"id": float64(1)})
		publisher.Publish(1, event)
	})
}

func TestNoOpPublisher_Implements_EventPublisher(t *testing.T) {
	// Compile-time check that NoOpPublisher implements EventPublisher
	var _ EventPublisher = (*NoOpPublisher)(nil)
}
