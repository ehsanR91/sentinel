package ws

import (
	"encoding/json"
	"sync"
)

// Message is a typed WebSocket envelope.
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub event loop (blocking — run in a goroutine).
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()

		case data := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- data:
				default:
					// Client send buffer full — drop and schedule unregister
					go func(cl *Client) { h.unregister <- cl }(c)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast enqueues a typed message for delivery to all clients.
func (h *Hub) Broadcast(msgType string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	data, err := json.Marshal(Message{Type: msgType, Payload: raw})
	if err != nil {
		return
	}
	h.broadcast <- data
}

// ClientCount returns the number of connected WebSocket clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
