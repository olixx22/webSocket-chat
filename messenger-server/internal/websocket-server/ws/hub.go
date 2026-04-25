package ws

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	log     *slog.Logger
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*Client]struct{}
}

func NewHub(log *slog.Logger) *Hub {
	return &Hub{
		log:     log.With(slog.String("component", "ws-hub")),
		clients: make(map[uuid.UUID]map[*Client]struct{}),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.chatID] == nil {
		h.clients[client.chatID] = make(map[*Client]struct{})
	}

	h.clients[client.chatID][client] = struct{}{}
	h.log.Info("client registered", slog.String("chat_id", client.chatID.String()), slog.String("user_id", client.userID.String()))
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clientsByChat := h.clients[client.chatID]
	if clientsByChat == nil {
		return
	}

	delete(clientsByChat, client)
	if len(clientsByChat) == 0 {
		delete(h.clients, client.chatID)
	}

	close(client.send)
	h.log.Info("client unregistered", slog.String("chat_id", client.chatID.String()), slog.String("user_id", client.userID.String()))
}

func (h *Hub) Broadcast(chatID uuid.UUID, event outboundEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		h.log.Error("failed to marshal websocket event", slog.Any("err", err))
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients[chatID] {
		select {
		case client.send <- payload:
		default:
			h.log.Warn("dropping websocket client with full send buffer", slog.String("chat_id", client.chatID.String()), slog.String("user_id", client.userID.String()))
		}
	}
}
