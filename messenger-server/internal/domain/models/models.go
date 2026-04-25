package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
    ID        uuid.UUID
    Username  string
    AvatarURL *string
    Bio       *string
}

type Chat struct {
    ID        uuid.UUID
    IsGroup   bool
    Title     string
    Members   []User
}

type ChatPreview struct {
	ID          uuid.UUID
	IsGroup     bool
	Title       string
	LastMessage *Message
}

type MemberRole string

const (
    RoleOwner MemberRole = "owner"
    RoleAdmin MemberRole = "admin"
    RoleMember MemberRole = "member"
)

type ChatMember struct {
    ChatID uuid.UUID
    UserID uuid.UUID
    Role   MemberRole
}

type MemberInput struct {
	UserID uuid.UUID
	Role MemberRole
}

type Message struct {
    ID        uuid.UUID
    ChatID    uuid.UUID
    SenderID  *uuid.UUID
    Content   *string
    Type      string
    CreatedAt time.Time
}

type MessageStatusType string

const (
    StatusSent     MessageStatusType = "sent"
    StatusDelivered MessageStatusType = "delivered"
    StatusRead     MessageStatusType = "read"
)

type MessageStatus struct {
    MessageID uuid.UUID
    UserID    uuid.UUID
    Status    MessageStatusType
}