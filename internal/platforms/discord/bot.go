package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Client represents a Discord bot client
type Client struct {
	session     *discordgo.Session
	token       string
	guildID     string
	isConnected bool
	webhooks    map[string]string // channelID -> webhookURL mapping
}

// NewClient creates a new Discord client
func NewClient(token, guildID string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("Discord bot token is required")
	}

	// Create a new Discord session using the provided bot token
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %v", err)
	}

	client := &Client{
		session:     session,
		token:       token,
		guildID:     guildID,
		isConnected: false,
		webhooks:    make(map[string]string),
	}

	return client, nil
}

// Connect connects to Discord
func (c *Client) Connect() error {
	if c.isConnected {
		return nil
	}

	// Open a websocket connection to Discord and begin listening
	err := c.session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection to Discord: %v", err)
	}

	c.isConnected = true
	log.Printf("✅ Discord bot connected successfully")

	return nil
}

// Disconnect disconnects from Discord
func (c *Client) Disconnect() error {
	if !c.isConnected {
		return nil
	}

	err := c.session.Close()
	if err != nil {
		log.Printf("❌ Error closing Discord connection: %v", err)
		return err
	}

	c.isConnected = false
	log.Printf("🔌 Discord bot disconnected")
	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.isConnected
}

// SendMessage sends a message to a Discord channel
func (c *Client) SendMessage(channelID, message string) error {
	if !c.isConnected {
		return fmt.Errorf("Discord client is not connected")
	}

	_, err := c.session.ChannelMessageSend(channelID, message)
	if err != nil {
		return fmt.Errorf("error sending message to Discord: %v", err)
	}

	return nil
}

// SendEmbed sends an embed message to a Discord channel
func (c *Client) SendEmbed(channelID string, embed *discordgo.MessageEmbed) error {
	if !c.isConnected {
		return fmt.Errorf("Discord client is not connected")
	}

	_, err := c.session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		return fmt.Errorf("error sending embed to Discord: %v", err)
	}

	return nil
}

// GetGuildChannels returns all channels in the configured guild
func (c *Client) GetGuildChannels() ([]*discordgo.Channel, error) {
	if !c.isConnected {
		return nil, fmt.Errorf("Discord client is not connected")
	}

	if c.guildID == "" {
		return nil, fmt.Errorf("guild ID not configured")
	}

	channels, err := c.session.GuildChannels(c.guildID)
	if err != nil {
		return nil, fmt.Errorf("error getting guild channels: %v", err)
	}

	return channels, nil
}

// GetChannel returns information about a specific channel
func (c *Client) GetChannel(channelID string) (*discordgo.Channel, error) {
	if !c.isConnected {
		return nil, fmt.Errorf("Discord client is not connected")
	}

	channel, err := c.session.Channel(channelID)
	if err != nil {
		return nil, fmt.Errorf("error getting channel info: %v", err)
	}

	return channel, nil
}

// RegisterCommands registers slash commands for the bot
func (c *Client) RegisterCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "bridge",
			Description: "Manage bridge connections",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "status",
					Description: "Show bridge status",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "create",
					Description: "Create a new bridge",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "platform",
							Description: "Target platform (telegram)",
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{
									Name:  "Telegram",
									Value: "telegram",
								},
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "room",
							Description: "Target room/chat ID",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a bridge",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "platform",
							Description: "Platform to remove bridge from",
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{
									Name:  "Telegram",
									Value: "telegram",
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "config",
			Description: "Bot configuration commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "platforms",
					Description: "Show enabled platforms",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "channels",
					Description: "List available channels",
				},
			},
		},
		{
			Name:        "help",
			Description: "Show bot help information",
		},
	}

	guildID := c.guildID
	if guildID == "" {
		// Register global commands if no guild specified
		guildID = ""
	}

	for _, command := range commands {
		_, err := c.session.ApplicationCommandCreate(c.session.State.User.ID, guildID, command)
		if err != nil {
			return fmt.Errorf("cannot create command %s: %v", command.Name, err)
		}
	}

	log.Printf("✅ Discord slash commands registered successfully")
	return nil
}

// SetMessageHandler sets the message create handler
func (c *Client) SetMessageHandler(handler func(*discordgo.Session, *discordgo.MessageCreate)) {
	c.session.AddHandler(handler)
}

// SetInteractionHandler sets the interaction create handler for slash commands
func (c *Client) SetInteractionHandler(handler func(*discordgo.Session, *discordgo.InteractionCreate)) {
	c.session.AddHandler(handler)
}

// SetReadyHandler sets the ready event handler
func (c *Client) SetReadyHandler(handler func(*discordgo.Session, *discordgo.Ready)) {
	c.session.AddHandler(handler)
}

// GetBotUser returns the bot user information
func (c *Client) GetBotUser() *discordgo.User {
	if c.session.State != nil {
		return c.session.State.User
	}
	return nil
}

// WebhookPayload represents a Discord webhook message payload
type WebhookPayload struct {
	Content   string `json:"content,omitempty"`
	Username  string `json:"username,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// GetOrCreateWebhook gets or creates a webhook for a channel
func (c *Client) GetOrCreateWebhook(channelID string) (string, error) {
	// Check if we already have a webhook for this channel
	if webhookURL, exists := c.webhooks[channelID]; exists {
		return webhookURL, nil
	}

	// Create a new webhook
	webhook, err := c.session.WebhookCreate(channelID, "Bridge Bot", "")
	if err != nil {
		return "", fmt.Errorf("failed to create webhook: %v", err)
	}

	webhookURL := fmt.Sprintf("https://discord.com/api/webhooks/%s/%s", webhook.ID, webhook.Token)
	c.webhooks[channelID] = webhookURL

	log.Printf("✅ Created Discord webhook for channel %s", channelID)
	return webhookURL, nil
}

// SendWebhookMessage sends a message via webhook with custom username and avatar
func (c *Client) SendWebhookMessage(channelID, content, username, avatarURL string) error {
	webhookURL, err := c.GetOrCreateWebhook(channelID)
	if err != nil {
		return fmt.Errorf("failed to get webhook: %v", err)
	}

	// Create webhook payload
	payload := WebhookPayload{
		Content:   content,
		Username:  username,
		AvatarURL: avatarURL,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %v", err)
	}

	// Send HTTP POST request to webhook URL
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook request failed with status: %d", resp.StatusCode)
	}

	log.Printf("✅ Webhook message sent to Discord channel %s", channelID)
	return nil
}

// GetPlatformAvatar returns avatar URL for different platforms
func (c *Client) GetPlatformAvatar(platform string) string {
	switch strings.ToLower(platform) {
	case "telegram":
		return "https://cdn4.iconfinder.com/data/icons/logos-and-brands/512/335_Telegram_logo-256.png"
	case "discord":
		return "https://cdn4.iconfinder.com/data/icons/logos-and-brands/512/91_Discord_logo_logos-256.png"
	default:
		return "https://cdn4.iconfinder.com/data/icons/ionicons/512/icon-chatbubble-working-256.png"
	}
}

// GetUserAvatar gets a user's avatar URL based on platform and user info
func (c *Client) GetUserAvatar(platform, userID, username string) string {
	switch strings.ToLower(platform) {
	case "telegram":
		// Telegram'da kullanıcı avatarını almak için API çağrısı gerekir
		// Şimdilik platform avatarını kullanıyoruz
		return c.GetPlatformAvatar("telegram")
	case "discord":
		// Discord'da kullanıcı avatarını almaya çalışalım
		if c.isConnected && userID != "" {
			if user, err := c.session.User(userID); err == nil {
				return user.AvatarURL("256")
			}
		}
		return c.GetPlatformAvatar("discord")
	default:
		return c.GetPlatformAvatar("unknown")
	}
}
