package handlers

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

type NotificationEvent struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
	Read      bool   `json:"read"`
	Avatar    string `json:"avatar,omitempty"`
	Link      string `json:"link,omitempty"`
}

type NotificationHub struct {
	mu sync.RWMutex

	// userID -> connected clients
	clients map[uuid.UUID]map[chan []byte]struct{}
}

func NewNotificationHub() *NotificationHub {
	return &NotificationHub{
		clients: make(map[uuid.UUID]map[chan []byte]struct{}),
	}
}

func (h *NotificationHub) AddClient(
	userID uuid.UUID,
	ch chan []byte,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[userID]; !exists {
		h.clients[userID] = make(map[chan []byte]struct{})
	}

	h.clients[userID][ch] = struct{}{}
}

func (h *NotificationHub) RemoveClient(
	userID uuid.UUID,
	ch chan []byte,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	userClients, exists := h.clients[userID]
	if !exists {
		return
	}

	if _, exists := userClients[ch]; !exists {
		return
	}

	delete(userClients, ch)

	close(ch)

	if len(userClients) == 0 {
		delete(h.clients, userID)
	}
}

func (h *NotificationHub) Publish(
	userID uuid.UUID,
	event NotificationEvent,
) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	userClients, exists := h.clients[userID]

	if !exists {
		return
	}

	for client := range userClients {
		select {
		case client <- data:
		default:
			// Slow client হলে server block করবে না
		}
	}
}