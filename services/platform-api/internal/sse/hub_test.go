package sse

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}

	if hub.clients == nil {
		t.Error("NewHub() should initialize clients map")
	}

	if len(hub.clients) != 0 {
		t.Error("NewHub() should create empty clients map")
	}
}

func TestSubscribe(t *testing.T) {
	t.Run("creates client with correct job ID", func(t *testing.T) {
		hub := NewHub()
		client := hub.Subscribe("job-123")

		if client.JobID != "job-123" {
			t.Errorf("client.JobID = %q, want %q", client.JobID, "job-123")
		}
	})

	t.Run("creates buffered events channel", func(t *testing.T) {
		hub := NewHub()
		client := hub.Subscribe("job-123")

		if client.Events == nil {
			t.Fatal("client.Events channel is nil")
		}

		if cap(client.Events) != 16 {
			t.Errorf("client.Events capacity = %d, want 16", cap(client.Events))
		}
	})

	t.Run("creates done channel", func(t *testing.T) {
		hub := NewHub()
		client := hub.Subscribe("job-123")

		if client.Done == nil {
			t.Error("client.Done channel is nil")
		}
	})

	t.Run("adds client to hub", func(t *testing.T) {
		hub := NewHub()
		client := hub.Subscribe("job-123")

		if hub.ClientCount("job-123") != 1 {
			t.Error("client not added to hub")
		}

		// Verify the specific client is tracked
		hub.mu.RLock()
		_, exists := hub.clients["job-123"][client]
		hub.mu.RUnlock()

		if !exists {
			t.Error("specific client not found in hub")
		}
	})

	t.Run("multiple clients for same job", func(t *testing.T) {
		hub := NewHub()
		client1 := hub.Subscribe("job-123")
		client2 := hub.Subscribe("job-123")

		if hub.ClientCount("job-123") != 2 {
			t.Errorf("ClientCount = %d, want 2", hub.ClientCount("job-123"))
		}

		if client1 == client2 {
			t.Error("should create distinct client instances")
		}
	})

	t.Run("multiple clients for different jobs", func(t *testing.T) {
		hub := NewHub()
		hub.Subscribe("job-1")
		hub.Subscribe("job-2")
		hub.Subscribe("job-2")

		if hub.ClientCount("job-1") != 1 {
			t.Errorf("ClientCount(job-1) = %d, want 1", hub.ClientCount("job-1"))
		}

		if hub.ClientCount("job-2") != 2 {
			t.Errorf("ClientCount(job-2) = %d, want 2", hub.ClientCount("job-2"))
		}
	})
}

func TestUnsubscribe(t *testing.T) {
	t.Run("removes client from hub", func(t *testing.T) {
		hub := NewHub()
		client := hub.Subscribe("job-123")

		hub.Unsubscribe(client)

		if hub.ClientCount("job-123") != 0 {
			t.Error("client not removed from hub")
		}
	})

	t.Run("closes events channel", func(t *testing.T) {
		hub := NewHub()
		client := hub.Subscribe("job-123")

		hub.Unsubscribe(client)

		// Reading from closed channel should return immediately
		select {
		case _, ok := <-client.Events:
			if ok {
				t.Error("expected channel to be closed")
			}
		default:
			t.Error("expected channel to be closed and readable")
		}
	})

	t.Run("cleans up empty job entry", func(t *testing.T) {
		hub := NewHub()
		client := hub.Subscribe("job-123")

		hub.Unsubscribe(client)

		hub.mu.RLock()
		_, exists := hub.clients["job-123"]
		hub.mu.RUnlock()

		if exists {
			t.Error("empty job entry should be removed from map")
		}
	})

	t.Run("preserves other clients for same job", func(t *testing.T) {
		hub := NewHub()
		client1 := hub.Subscribe("job-123")
		hub.Subscribe("job-123")

		hub.Unsubscribe(client1)

		if hub.ClientCount("job-123") != 1 {
			t.Errorf("ClientCount = %d, want 1", hub.ClientCount("job-123"))
		}
	})

	t.Run("handles unsubscribe of non-existent client", func(_ *testing.T) {
		hub := NewHub()
		client := &Client{
			JobID:  "non-existent",
			Events: make(chan []byte, 16),
			Done:   make(chan struct{}),
		}

		// Should not panic
		hub.Unsubscribe(client)
	})
}

func TestBroadcast(t *testing.T) {
	t.Run("sends event to all clients for a job", testBroadcastSendsEventToAllClients)
	t.Run("does not send to clients of different job", testBroadcastSkipsOtherJobs)
	t.Run("handles non-existent job", testBroadcastHandlesNonExistentJob)
	t.Run("evicts oldest on full client buffer", testBroadcastEvictsOldestOnFullBuffer)
	t.Run("handles json marshal error", testBroadcastHandlesMarshalError)
}

func testBroadcastSendsEventToAllClients(t *testing.T) {
	hub := NewHub()
	client1 := hub.Subscribe("job-123")
	client2 := hub.Subscribe("job-123")

	event := map[string]string{"type": "status", "state": "running"}
	hub.Broadcast("job-123", event)

	// Give time for broadcast
	time.Sleep(10 * time.Millisecond)

	expected, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	assertClientReceivesEvent(t, client1, expected, "client1")
	assertClientReceivesEvent(t, client2, expected, "client2")
}

func testBroadcastSkipsOtherJobs(t *testing.T) {
	hub := NewHub()
	client1 := hub.Subscribe("job-1")
	client2 := hub.Subscribe("job-2")

	hub.Broadcast("job-1", map[string]string{"msg": "hello"})

	time.Sleep(10 * time.Millisecond)

	assertClientReceivesAny(t, client1, "client1 should receive event")
	assertClientReceivesNone(t, client2, "client2 should not receive event for different job")
}

func testBroadcastHandlesNonExistentJob(t *testing.T) {
	t.Helper()

	hub := NewHub()

	// Should not panic
	hub.Broadcast("non-existent", map[string]string{"msg": "hello"})
}

func testBroadcastEvictsOldestOnFullBuffer(t *testing.T) {
	hub := NewHub()
	client := hub.Subscribe("job-123")

	// Fill the buffer (capacity is 16)
	for i := range 16 {
		hub.Broadcast("job-123", map[string]int{"i": i})
	}

	// One more event should evict the oldest entry.
	hub.Broadcast("job-123", map[string]int{"i": 16})

	time.Sleep(10 * time.Millisecond)

	// Should still have 16 events (buffer size) and include the newest value.
	seen := make(map[int]struct{})
	count := 0
	draining := true

	for draining {
		select {
		case data := <-client.Events:
			var payload map[string]int
			if err := json.Unmarshal(data, &payload); err == nil {
				if v, ok := payload["i"]; ok {
					seen[v] = struct{}{}
				}
			}

			count++
		default:
			draining = false
		}
	}

	if count != 16 {
		t.Errorf("received %d events, want 16 (buffer size)", count)
	}

	if _, ok := seen[16]; !ok {
		t.Errorf("expected newest event to be retained")
	}

	if _, ok := seen[0]; ok {
		t.Errorf("expected oldest event to be evicted")
	}
}

func testBroadcastHandlesMarshalError(t *testing.T) {
	t.Helper()

	hub := NewHub()
	hub.Subscribe("job-123")

	// Create an unmarshalable value (channel)
	hub.Broadcast("job-123", make(chan int))

	// Should not panic and client should receive nothing
	time.Sleep(10 * time.Millisecond)
}

func assertClientReceivesEvent(t *testing.T, client *Client, expected []byte, label string) {
	t.Helper()

	select {
	case data := <-client.Events:
		if !bytes.Equal(data, expected) {
			t.Errorf("%s received %s, want %s", label, data, expected)
		}
	default:
		t.Errorf("%s did not receive event", label)
	}
}

func assertClientReceivesAny(t *testing.T, client *Client, message string) {
	t.Helper()

	select {
	case <-client.Events:
		// Expected
	default:
		t.Error(message)
	}
}

func assertClientReceivesNone(t *testing.T, client *Client, message string) {
	t.Helper()

	select {
	case <-client.Events:
		t.Error(message)
	default:
		// Expected
	}
}

func TestClientCount(t *testing.T) {
	t.Run("returns 0 for non-existent job", func(t *testing.T) {
		hub := NewHub()

		if count := hub.ClientCount("non-existent"); count != 0 {
			t.Errorf("ClientCount = %d, want 0", count)
		}
	})

	t.Run("returns correct count", func(t *testing.T) {
		hub := NewHub()
		hub.Subscribe("job-123")
		hub.Subscribe("job-123")
		hub.Subscribe("job-123")

		if count := hub.ClientCount("job-123"); count != 3 {
			t.Errorf("ClientCount = %d, want 3", count)
		}
	})
}

func TestTotalClients(t *testing.T) {
	t.Run("returns 0 for empty hub", func(t *testing.T) {
		hub := NewHub()

		if total := hub.TotalClients(); total != 0 {
			t.Errorf("TotalClients = %d, want 0", total)
		}
	})

	t.Run("returns correct total across jobs", func(t *testing.T) {
		hub := NewHub()
		hub.Subscribe("job-1")
		hub.Subscribe("job-1")
		hub.Subscribe("job-2")
		hub.Subscribe("job-3")
		hub.Subscribe("job-3")
		hub.Subscribe("job-3")

		if total := hub.TotalClients(); total != 6 {
			t.Errorf("TotalClients = %d, want 6", total)
		}
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent subscribe/unsubscribe", testConcurrentSubscribeUnsubscribe)
	t.Run("concurrent broadcast", testConcurrentBroadcast)
	t.Run("concurrent subscribe and broadcast", testConcurrentSubscribeAndBroadcast)
}

func testConcurrentSubscribeUnsubscribe(t *testing.T) {
	hub := NewHub()

	var wg sync.WaitGroup

	clients := make(chan *Client, 100)

	// Concurrent subscriptions
	for range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			client := hub.Subscribe("job-123")
			clients <- client
		}()
	}

	wg.Wait()
	close(clients)

	if count := hub.ClientCount("job-123"); count != 50 {
		t.Errorf("ClientCount = %d, want 50", count)
	}

	// Concurrent unsubscriptions
	for client := range clients {
		wg.Add(1)

		go func(c *Client) {
			defer wg.Done()

			hub.Unsubscribe(c)
		}(client)
	}

	wg.Wait()

	if count := hub.ClientCount("job-123"); count != 0 {
		t.Errorf("ClientCount after unsubscribe = %d, want 0", count)
	}
}

func testConcurrentBroadcast(t *testing.T) {
	hub := NewHub()
	client := hub.Subscribe("job-123")

	var wg sync.WaitGroup

	// Start broadcasting concurrently
	for i := range 10 {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			hub.Broadcast("job-123", map[string]int{"i": i})
		}(i)
	}

	wg.Wait()

	// Drain events
	time.Sleep(10 * time.Millisecond)

	count := 0
	draining := true

	for draining {
		select {
		case <-client.Events:
			count++
		default:
			draining = false
		}
	}

	if count != 10 {
		t.Errorf("received %d events, want 10", count)
	}
}

func testConcurrentSubscribeAndBroadcast(t *testing.T) {
	hub := NewHub()

	var wg sync.WaitGroup

	clients := make([]*Client, 0, 20)

	var mu sync.Mutex

	// Subscribe and broadcast concurrently
	for i := range 20 {
		wg.Add(2)

		go func() {
			defer wg.Done()

			c := hub.Subscribe("job-123")

			mu.Lock()

			clients = append(clients, c)

			mu.Unlock()
		}()

		go func(i int) {
			defer wg.Done()

			hub.Broadcast("job-123", map[string]int{"i": i})
		}(i)
	}

	wg.Wait()

	// Should not have panicked
	if len(clients) != 20 {
		t.Errorf("created %d clients, want 20", len(clients))
	}
}
