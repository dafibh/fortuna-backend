package websocket

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient is a test double for Client that captures sent messages
type mockClient struct {
	id          string
	workspaceID int32
	messages    [][]byte
	mu          sync.Mutex
	closed      bool
}

func newMockClient(id string, workspaceID int32) *mockClient {
	return &mockClient{
		id:          id,
		workspaceID: workspaceID,
		messages:    make([][]byte, 0),
	}
}

func (m *mockClient) ID() string {
	return m.id
}

func (m *mockClient) WorkspaceID() int32 {
	return m.workspaceID
}

func (m *mockClient) Send(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClientClosed
	}
	m.messages = append(m.messages, data)
	return nil
}

func (m *mockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockClient) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *mockClient) GetMessages() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([][]byte, len(m.messages))
	copy(copied, m.messages)
	return copied
}

func TestHub_RegisterUnregister(t *testing.T) {
	hub := NewHub()

	client1 := newMockClient("client-1", 1)
	client2 := newMockClient("client-2", 1)
	client3 := newMockClient("client-3", 2)

	// Register clients
	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)

	// Verify counts
	assert.Equal(t, 2, hub.ClientCount(1))
	assert.Equal(t, 1, hub.ClientCount(2))
	assert.Equal(t, 0, hub.ClientCount(999))

	// Unregister one client from workspace 1
	hub.Unregister(client1)
	assert.Equal(t, 1, hub.ClientCount(1))

	// Unregister remaining clients
	hub.Unregister(client2)
	hub.Unregister(client3)
	assert.Equal(t, 0, hub.ClientCount(1))
	assert.Equal(t, 0, hub.ClientCount(2))
}

func TestHub_Broadcast_WorkspaceIsolation(t *testing.T) {
	hub := NewHub()

	// Clients in workspace 1
	client1a := newMockClient("client-1a", 1)
	client1b := newMockClient("client-1b", 1)

	// Client in workspace 2
	client2 := newMockClient("client-2", 2)

	hub.Register(client1a)
	hub.Register(client1b)
	hub.Register(client2)

	// Broadcast to workspace 1
	evt := TransactionCreated(map[string]interface{}{"id": float64(42)})
	hub.Broadcast(1, evt)

	// Give goroutines time to process
	time.Sleep(10 * time.Millisecond)

	// Workspace 1 clients should receive the message
	msgs1a := client1a.GetMessages()
	msgs1b := client1b.GetMessages()
	assert.Len(t, msgs1a, 1, "client1a should receive 1 message")
	assert.Len(t, msgs1b, 1, "client1b should receive 1 message")

	// Workspace 2 client should NOT receive the message
	msgs2 := client2.GetMessages()
	assert.Len(t, msgs2, 0, "client2 should not receive message from workspace 1")
}

func TestHub_Broadcast_MultipleFanOut(t *testing.T) {
	hub := NewHub()

	// Create multiple clients in the same workspace
	clients := make([]*mockClient, 5)
	for i := 0; i < 5; i++ {
		clients[i] = newMockClient("client-"+string(rune('a'+i)), 1)
		hub.Register(clients[i])
	}

	// Broadcast event
	evt := TransactionUpdated(map[string]interface{}{"id": float64(1)})
	hub.Broadcast(1, evt)

	// Give goroutines time to process
	time.Sleep(10 * time.Millisecond)

	// All clients should receive the message
	for i, c := range clients {
		msgs := c.GetMessages()
		assert.Len(t, msgs, 1, "client %d should receive message", i)
	}
}

func TestHub_ConcurrentAccess(t *testing.T) {
	hub := NewHub()

	var wg sync.WaitGroup
	clientCount := 50

	// Concurrently register clients
	clients := make([]*mockClient, clientCount)
	for i := 0; i < clientCount; i++ {
		clients[i] = newMockClient("client-"+string(rune(i)), int32(i%5))
	}

	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			hub.Register(clients[idx])
		}(i)
	}

	wg.Wait()

	// Verify total is correct (10 per workspace, 5 workspaces)
	total := 0
	for ws := int32(0); ws < 5; ws++ {
		total += hub.ClientCount(ws)
	}
	assert.Equal(t, clientCount, total)

	// Concurrently broadcast and unregister
	for i := 0; i < clientCount; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			evt := TransactionCreated(map[string]interface{}{"id": float64(idx)})
			hub.Broadcast(int32(idx%5), evt)
		}(i)
		go func(idx int) {
			defer wg.Done()
			hub.Unregister(clients[idx])
		}(i)
	}

	wg.Wait()

	// After unregistering all, counts should be 0
	for ws := int32(0); ws < 5; ws++ {
		assert.Equal(t, 0, hub.ClientCount(ws))
	}
}

func TestHub_UnregisterNonexistent(t *testing.T) {
	hub := NewHub()

	client := newMockClient("client-1", 1)

	// Should not panic when unregistering a client that was never registered
	require.NotPanics(t, func() {
		hub.Unregister(client)
	})
}

func TestHub_BroadcastToEmptyWorkspace(t *testing.T) {
	hub := NewHub()

	// Should not panic when broadcasting to workspace with no clients
	require.NotPanics(t, func() {
		evt := TransactionCreated(map[string]interface{}{"id": float64(1)})
		hub.Broadcast(999, evt)
	})
}
