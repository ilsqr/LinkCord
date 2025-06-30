package types

import "time"

// Platform constants
const (
	PlatformDiscord  = "discord"
	PlatformTelegram = "telegram"
)

// MessageType constants
const (
	MessageTypeText  = "text"
	MessageTypeImage = "image"
	MessageTypeFile  = "file"
)

// BridgeMessage represents a message that needs to be bridged
type BridgeMessage struct {
	ID              string    `json:"id"`
	SourcePlatform  string    `json:"source_platform"`
	SourceChannelID string    `json:"source_channel_id"`
	SourceUserID    string    `json:"source_user_id"`
	Username        string    `json:"username"`
	Content         string    `json:"content"`
	MessageType     string    `json:"message_type"`
	Timestamp       time.Time `json:"timestamp"`
	Attachments     []string  `json:"attachments,omitempty"`
}

// BridgeConnection represents a bridge between two platforms
type BridgeConnection struct {
	ID              string    `json:"id"`
	SourcePlatform  string    `json:"source_platform"`
	SourceChannelID string    `json:"source_channel_id"`
	TargetPlatform  string    `json:"target_platform"`
	TargetChannelID string    `json:"target_channel_id"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
}

// Platform interface defines methods that each platform must implement
type Platform interface {
	GetName() string
	IsConnected() bool
	SendMessage(channelID, content string) error
	FormatMessage(message *BridgeMessage) string
}

// BridgeCore interface for managing bridges
type BridgeCore interface {
	RegisterPlatform(platform Platform)
	AddBridge(sourcePlatform, sourceChannelID, targetPlatform, targetChannelID string) error
	RemoveBridge(sourceChannelID, targetPlatform string) error
	GetBridges(channelID string) []*BridgeConnection
	GetPlatformStatus() map[string]bool
	ProcessMessage(message *BridgeMessage) error
	SetUserMapping(platform, userID, displayName string)
}
