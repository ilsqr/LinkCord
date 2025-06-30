package bridge

import (
	"fmt"
	"strings"

	"dcbot/internal/platforms/discord"
	"dcbot/internal/types"
)

// DiscordAdapter implements the Platform interface for Discord
type DiscordAdapter struct {
	client *discord.Client
}

// NewDiscordAdapter creates a new Discord adapter
func NewDiscordAdapter(client *discord.Client) *DiscordAdapter {
	return &DiscordAdapter{
		client: client,
	}
}

// GetName returns the platform name
func (da *DiscordAdapter) GetName() string {
	return types.PlatformDiscord
}

// IsConnected returns whether the Discord client is connected
func (da *DiscordAdapter) IsConnected() bool {
	return da.client.IsConnected()
}

// SendMessage sends a message to a Discord channel using webhook
func (da *DiscordAdapter) SendMessage(channelID, content string) error {
	// Try to send as regular message if no formatting is needed
	return da.client.SendMessage(channelID, content)
}

// SendBridgeMessage sends a bridge message using webhook for better formatting
func (da *DiscordAdapter) SendBridgeMessage(channelID string, message *types.BridgeMessage) error {
	// Clean and format username
	username := message.Username
	if username == "" {
		username = "Anonymous"
	}
	
	// Remove @ symbol for webhook username (Discord adds it automatically)
	if strings.HasPrefix(username, "@") {
		username = username[1:]
	}
	
	// Add platform prefix to username
	var platformPrefix string
	switch message.SourcePlatform {
	case types.PlatformTelegram:
		platformPrefix = "[TELEGRAM] "
		username = platformPrefix + username
	default:
		platformPrefix = "[BRIDGE] "
		username = platformPrefix + username
	}
	
	// Get user-specific avatar if possible, fallback to platform avatar
	avatarURL := da.client.GetUserAvatar(message.SourcePlatform, message.SourceUserID, message.Username)
	
	// Send via webhook
	return da.client.SendWebhookMessage(channelID, message.Content, username, avatarURL)
}

// FormatMessage formats a bridge message for Discord (fallback method)
func (da *DiscordAdapter) FormatMessage(message *types.BridgeMessage) string {
	// Use [PLATFORM] format for consistency
	var platformPrefix string
	switch message.SourcePlatform {
	case types.PlatformTelegram:
		platformPrefix = "[TELEGRAM]"
	case types.PlatformDiscord:
		platformPrefix = "[DISCORD]"
	default:
		platformPrefix = "[BRIDGE]"
	}

	// Format username (preserve Telegram @ format if present)
	username := message.Username
	if username == "" {
		username = "anonymous"
	}
	
	// Format the message for Discord
	formattedMessage := fmt.Sprintf("%s **%s**: %s", platformPrefix, username, message.Content)
	
	return formattedMessage
}

// cleanUsername cleans a username to be Discord-safe
func cleanUsername(username string) string {
	// Remove Discord mention syntax and other problematic characters
	username = strings.ReplaceAll(username, "@", "")
	username = strings.ReplaceAll(username, "#", "")
	username = strings.ReplaceAll(username, "`", "")
	username = strings.ReplaceAll(username, "*", "")
	username = strings.ReplaceAll(username, "_", "")
	username = strings.ReplaceAll(username, "~", "")
	
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
