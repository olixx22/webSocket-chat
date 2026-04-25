package ws

import "ws_chat/messenger-server/internal/domain/models"

const (
	eventTypePing           = "ping"
	eventTypePong           = "pong"
	eventTypeError          = "error"
	eventTypeMessageCreate  = "message.create"
	eventTypeMessageCreated = "message.created"
)

type inboundEvent struct {
	Type        string `json:"type"`
	Content     string `json:"content,omitempty"`
	MessageType string `json:"message_type,omitempty"`
}

type outboundEvent struct {
	Type    string          `json:"type"`
	Message *models.Message `json:"message,omitempty"`
	Error   string          `json:"error,omitempty"`
}
