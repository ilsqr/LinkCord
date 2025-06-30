package models

import (
	"time"
)

// User represents a user across different platforms
type User struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// UserMapping represents the mapping between users on different platforms
type UserMapping struct {
	ID             int       `db:"id" json:"id"`
	UserID         int       `db:"user_id" json:"user_id"`
	Platform       string    `db:"platform" json:"platform"`       // "telegram", "discord"
	PlatformUserID string    `db:"platform_user_id" json:"platform_user_id"`
	Username       string    `db:"username" json:"username"`
	DisplayName    string    `db:"display_name" json:"display_name"`
	AvatarURL      string    `db:"avatar_url" json:"avatar_url"`
	IsActive       bool      `db:"is_active" json:"is_active"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// Room represents a chat room/channel/group across platforms
type Room struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// RoomMapping represents the mapping between rooms on different platforms
type RoomMapping struct {
	ID             int       `db:"id" json:"id"`
	RoomID         int       `db:"room_id" json:"room_id"`
	Platform       string    `db:"platform" json:"platform"`        // "telegram", "discord"
	PlatformRoomID string    `db:"platform_room_id" json:"platform_room_id"`
	RoomName       string    `db:"room_name" json:"room_name"`
	RoomType       string    `db:"room_type" json:"room_type"`       // "channel", "group", "dm"
	IsActive       bool      `db:"is_active" json:"is_active"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// Message represents a bridged message
type Message struct {
	ID              int       `db:"id" json:"id"`
	OriginalID      string    `db:"original_id" json:"original_id"`          // Original message ID from source platform
	SourcePlatform  string    `db:"source_platform" json:"source_platform"`  // "telegram", "discord"
	SourceRoomID    string    `db:"source_room_id" json:"source_room_id"`
	SourceUserID    string    `db:"source_user_id" json:"source_user_id"`
	Content         string    `db:"content" json:"content"`
	MessageType     string    `db:"message_type" json:"message_type"`         // "text", "image", "file", "audio", "video"
	MediaURL        string    `db:"media_url" json:"media_url"`
	MediaMimeType   string    `db:"media_mime_type" json:"media_mime_type"`
	ReplyToID       *int      `db:"reply_to_id" json:"reply_to_id"`           // Reference to another message
	IsEdited        bool      `db:"is_edited" json:"is_edited"`
	IsDeleted       bool      `db:"is_deleted" json:"is_deleted"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

// MessageMapping represents how a message is mapped across platforms
type MessageMapping struct {
	ID             int       `db:"id" json:"id"`
	MessageID      int       `db:"message_id" json:"message_id"`
	Platform       string    `db:"platform" json:"platform"`
	PlatformMsgID  string    `db:"platform_msg_id" json:"platform_msg_id"`
	PlatformRoomID string    `db:"platform_room_id" json:"platform_room_id"`
	Status         string    `db:"status" json:"status"`                     // "sent", "failed", "pending"
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// BridgeConfig represents bridge configuration for room mappings
type BridgeConfig struct {
	ID               int       `db:"id" json:"id"`
	RoomID           int       `db:"room_id" json:"room_id"`
	IsActive         bool      `db:"is_active" json:"is_active"`
	AllowMedia       bool      `db:"allow_media" json:"allow_media"`
	AllowEdits       bool      `db:"allow_edits" json:"allow_edits"`
	AllowDeletes     bool      `db:"allow_deletes" json:"allow_deletes"`
	FilterWords      string    `db:"filter_words" json:"filter_words"`        // JSON array of filtered words
	MaxMessageLength int       `db:"max_message_length" json:"max_message_length"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}
