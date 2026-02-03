// Package sse provides Server-Sent Events support for real-time job status updates.
package sse

import (
	"encoding/json"
	"sync"
)

// Client represents a connected SSE client subscribed to a specific job.
type Client struct {
	JobID  string
	Events chan []byte
	Done   chan struct{}
}

// Hub manages SSE connections per job for broadcasting status updates.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{} // jobID -> clients
}

// NewHub creates a new SSE hub for managing client connections.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]struct{}),
	}
}

// Subscribe creates a new client subscription for a job.
func (h *Hub) Subscribe(jobID string) *Client {
	h.mu.Lock()
	defer h.mu.Unlock()

	client := &Client{
		JobID:  jobID,
		Events: make(chan []byte, 16),
		Done:   make(chan struct{}),
	}

	if h.clients[jobID] == nil {
		h.clients[jobID] = make(map[*Client]struct{})
	}

	h.clients[jobID][client] = struct{}{}

	return client
}

// Unsubscribe removes a client from the hub and closes its event channel.
func (h *Hub) Unsubscribe(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.JobID]; ok {
		delete(clients, client)

		if len(clients) == 0 {
			delete(h.clients, client.JobID)
		}
	}

	close(client.Events)
}

// Broadcast sends an event to all clients subscribed to a job.
func (h *Hub) Broadcast(jobID string, event any) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := h.clients[jobID]
	if len(clients) == 0 {
		return
	}

	for client := range clients {
		sendWithEviction(client.Events, data)
	}
}

func sendWithEviction(ch chan []byte, data []byte) {
	select {
	case ch <- data:
		return
	default:
	}

	// Buffer full: drop the oldest event to preserve the latest update.
	select {
	case <-ch:
	default:
	}

	select {
	case ch <- data:
	default:
		// If the channel is still full, drop the event.
	}
}

// ClientCount returns the number of connected clients for a job.
func (h *Hub) ClientCount(jobID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[jobID]; ok {
		return len(clients)
	}

	return 0
}

// TotalClients returns the total number of connected clients across all jobs.
func (h *Hub) TotalClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, clients := range h.clients {
		total += len(clients)
	}

	return total
}
