package bridge

import (
	"fmt"
	"strings"

	"dcbot/internal/platforms/telegram"
	"dcbot/internal/types"
)

// TelegramAdapter implements the Platform interface for Telegram
type TelegramAdapter struct {
	client *telegram.Client
}

// NewTelegramAdapter creates a new Telegram adapter
func NewTelegramAdapter(client *telegram.Client) *TelegramAdapter {
	return &TelegramAdapter{
		client: client,
	}
}

// GetName returns the platform name
func (ta *TelegramAdapter) GetName() string {
	return types.PlatformTelegram
}

// IsConnected returns whether the Telegram client is connected
func (ta *TelegramAdapter) IsConnected() bool {
	return ta.client.IsRunning()
}

// SendMessage sends a message to a Telegram chat
func (ta *TelegramAdapter) SendMessage(chatID, content string) error {
	return ta.client.SendMessage(chatID, content)
}

// FormatMessage formats a bridge message for Telegram
func (ta *TelegramAdapter) FormatMessage(message *types.BridgeMessage) string {
	// Use [PLATFORM] format instead of emojis
	var platformPrefix string
	switch message.SourcePlatform {
	case types.PlatformDiscord:
		platformPrefix = "[DISCORD]"
	case types.PlatformTelegram:
		platformPrefix = "[TELEGRAM]"
	default:
		platformPrefix = "[BRIDGE]"
	}

	// Use Telegram username format (@username)
	username := message.Username
	if !strings.HasPrefix(username, "@") && username != "" {
		username = "@" + username
	}
	if username == "@" || username == "" {
		username = "@anonymous"
	}
	
	// Format the message for Telegram
	formattedMessage := fmt.Sprintf("%s %s: %s", platformPrefix, username, message.Content)
	
	return formattedMessage
}

// cleanUsernameTelegram cleans a username to be Telegram-safe
func cleanUsernameTelegram(username string) string {
	// Remove Telegram username syntax and markdown characters
	username = strings.ReplaceAll(username, "@", "")
	username = strings.ReplaceAll(username, "*", "")
	username = strings.ReplaceAll(username, "_", "")
	username = strings.ReplaceAll(username, "`", "")
	username = strings.ReplaceAll(username, "[", "")
	username = strings.ReplaceAll(username, "]", "")
	
	// Limit length
	if len(username) > 32 {
		username = username[:29] + "..."
	}
	
	// Fallback if username becomes empty
	if username == "" {
		username = "Unknown"
	}
	
	return username
}
