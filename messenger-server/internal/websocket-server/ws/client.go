package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
	"ws_chat/messenger-server/internal/domain/models"
	"ws_chat/messenger-server/internal/storage"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8 * 1024
	sendBufferSize = 64
)

type messageStorage interface {
	CreateMessage(context.Context, models.Message) (models.Message, error)
}

type Client struct {
	log     *slog.Logger
	hub     *Hub
	storage messageStorage
	conn    *websocket.Conn
	send    chan []byte
	userID  uuid.UUID
	chatID  uuid.UUID
}

func NewClient(log *slog.Logger, hub *Hub, storage messageStorage, conn *websocket.Conn, userID, chatID uuid.UUID) *Client {
	return &Client{
		log:     log.With(slog.String("component", "ws-client")),
		hub:     hub,
		storage: storage,
		conn:    conn,
		send:    make(chan []byte, sendBufferSize),
		userID:  userID,
		chatID:  chatID,
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		var inbound inboundEvent
		if err := c.conn.ReadJSON(&inbound); err != nil {
			c.log.Info("websocket read loop stopped", slog.Any("err", err))
			return
		}

		if inbound.Type == eventTypePing {
			c.enqueue(outboundEvent{Type: eventTypePong})
			continue
		}

		if inbound.Type != eventTypeMessageCreate {
			c.enqueueError("unsupported event type")
			continue
		}

		if inbound.Content == "" {
			c.enqueueError("content is required")
			continue
		}

		msgType := inbound.MessageType
		if msgType == "" {
			msgType = "text"
		}

		senderID := c.userID
		content := inbound.Content

		created, err := c.storage.CreateMessage(context.Background(), models.Message{
			ID:       uuid.New(),
			ChatID:   c.chatID,
			SenderID: &senderID,
			Content:  &content,
			Type:     msgType,
		})
		if err != nil {
			if err == storage.ErrUserIsNotMember {
				c.enqueueError(err.Error())
				continue
			}
			c.log.Error("failed to persist websocket message", slog.Any("err", err))
			c.enqueueError("failed to persist message")
			continue
		}

		c.hub.Broadcast(c.chatID, outboundEvent{
			Type:    eventTypeMessageCreated,
			Message: &created,
		})
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.log.Info("websocket write loop stopped", slog.Any("err", err))
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.log.Info("websocket ping failed", slog.Any("err", err))
				return
			}
		}
	}
}

func (c *Client) enqueue(event outboundEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		c.log.Error("failed to marshal outbound websocket event", slog.Any("err", err))
		return
	}

	select {
	case c.send <- payload:
	default:
		c.log.Warn("websocket send buffer is full", slog.String("chat_id", c.chatID.String()), slog.String("user_id", c.userID.String()))
	}
}

func (c *Client) enqueueError(message string) {
	c.enqueue(outboundEvent{
		Type:  eventTypeError,
		Error: message,
	})
}
