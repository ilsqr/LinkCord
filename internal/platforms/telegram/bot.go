package telegram

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client struct {
	bot         *tgbotapi.BotAPI
	chatID      int64
	isRunning   bool
	stopChan    chan struct{}
	updatesChan tgbotapi.UpdatesChannel
}

type Config struct {
	BotToken string
	ChatID   string
}

// NewClient creates a new Telegram bot client
func NewClient(cfg Config) (*Client, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %v", err)
	}

	// Parse chat ID
	chatID, err := strconv.ParseInt(cfg.ChatID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid Telegram chat ID: %v", err)
	}

	log.Printf("✅ Telegram bot authorized: %s", bot.Self.UserName)

	client := &Client{
		bot:      bot,
		chatID:   chatID,
		stopChan: make(chan struct{}),
	}

	return client, nil
}

// Start begins listening for Telegram updates
func (c *Client) Start(messageHandler func(platform, chatID, userID, messageType, content string) error) error {
	if c.isRunning {
		return fmt.Errorf("Telegram client is already running")
	}

	// Store message handler callback
	messageHandlerCallback = messageHandler

	log.Printf("🚀 Starting Telegram bot...")
	log.Printf("📱 Bot username: @%s", c.bot.Self.UserName)
	log.Printf("📱 Monitoring chat ID: %d", c.chatID)

	// Delete webhook first to ensure polling works
	deleteWebhookConfig := tgbotapi.DeleteWebhookConfig{
		DropPendingUpdates: true,
	}
	_, err := c.bot.Request(deleteWebhookConfig)
	if err != nil {
		log.Printf("⚠️ Warning: Could not delete webhook: %v", err)
	} else {
		log.Printf("✅ Webhook deleted, using polling")
	}

	// Configure updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	log.Printf("🔄 Getting updates channel...")
	// Get updates channel
	c.updatesChan = c.bot.GetUpdatesChan(u)

	// Start processing updates in a goroutine
	go func() {
		log.Printf("📡 Starting update listener goroutine...")
		for {
			select {
			case update := <-c.updatesChan:
				log.Printf("📥 Received Telegram update: %+v", update)
				c.handleUpdate(update, messageHandler)
			case <-c.stopChan:
				log.Printf("🛑 Telegram update listener stopped")
				return
			}
		}
	}()

	c.isRunning = true
	log.Println("✅ Telegram bot started and listening for updates")
	return nil
}

// handleUpdate processes incoming Telegram updates
func (c *Client) handleUpdate(update tgbotapi.Update, messageHandler func(string, string, string, string, string) error) {
	log.Printf("🔍 Processing update: %+v", update)
	
	// Handle messages
	if update.Message != nil {
		message := update.Message
		log.Printf("📨 Message received - Chat ID: %d, User: %s, Text: %s", message.Chat.ID, message.From.UserName, message.Text)

		// Check if this is the monitored chat
		if message.Chat.ID != c.chatID {
			log.Printf("⏭️ Ignoring message from different chat (Expected: %d, Got: %d)", c.chatID, message.Chat.ID)
			return
		}

		// Skip messages from bots (including ourselves)
		if message.From.IsBot {
			log.Printf("🤖 Ignoring bot message from: %s", message.From.UserName)
			return
		}

		// Skip messages that look like bridge messages to prevent loops
		if strings.Contains(message.Text, "[DISCORD]") {
			log.Printf("⏭️ Ignoring potential bridge message: %s", message.Text)
			return
		}

		// Extract message information
		chatID := strconv.FormatInt(message.Chat.ID, 10)
		userID := strconv.FormatInt(message.From.ID, 10)
		
		// Get username - prioritize Telegram username over first name
		username := ""
		if message.From.UserName != "" {
			username = message.From.UserName // Telegram @username (without @)
		} else if message.From.FirstName != "" {
			username = message.From.FirstName
			if message.From.LastName != "" {
				username += " " + message.From.LastName
			}
		} else {
			username = "User" + userID // Fallback to User + ID
		}

		log.Printf("📨 Telegram user info - ID: %s, Username: %s, FirstName: %s, LastName: %s", 
			userID, message.From.UserName, message.From.FirstName, message.From.LastName)
		
		// Store user mapping for bridge core
		c.storeUserMapping(userID, username)

		var messageType string
		var content string

		// Determine message type and content
		switch {
		case message.Text != "":
			messageType = "text"
			content = message.Text

			// Handle bot commands
			if strings.HasPrefix(content, "/") {
				c.handleCommand(message)
				return
			}

		case message.Photo != nil:
			messageType = "image"
			content = message.Caption
			if content == "" {
				content = "📷 Image"
			}
			// TODO: Add photo URL/file handling

		case message.Document != nil:
			messageType = "file"
			content = message.Document.FileName
			if message.Caption != "" {
				content += ": " + message.Caption
			}

		case message.Audio != nil:
			messageType = "audio"
			content = "🎵 Audio"
			if message.Caption != "" {
				content += ": " + message.Caption
			}

		case message.Video != nil:
			messageType = "video"
			content = "🎥 Video"
			if message.Caption != "" {
				content += ": " + message.Caption
			}

		case message.Voice != nil:
			messageType = "audio"
			content = "🎤 Voice message"

		case message.Sticker != nil:
			messageType = "sticker"
			content = "🎨 " + message.Sticker.Emoji + " Sticker"

		default:
			messageType = "text"
			content = "📎 Unsupported message type"
		}

		log.Printf("📨 Telegram message from %s (%s): %s", username, userID, content)

		// Bridge the message to other platforms
		if messageHandler != nil {
			err := messageHandler("telegram", chatID, userID, messageType, content)
			if err != nil {
				log.Printf("❌ Failed to bridge Telegram message: %v", err)
			} else {
				log.Printf("✅ Telegram message bridged successfully")
			}
		}
	}

	// Handle callback queries (inline button presses)
	if update.CallbackQuery != nil {
		// Acknowledge the callback query
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
		c.bot.Request(callback)

		log.Printf("🔘 Telegram callback: %s", update.CallbackQuery.Data)
	}
}

// handleCommand processes bot commands
func (c *Client) handleCommand(message *tgbotapi.Message) {
	command := strings.Split(message.Text, " ")[0]
	_ = "" // args placeholder for future use
	if len(strings.Split(message.Text, " ")) > 1 {
		_ = strings.Join(strings.Split(message.Text, " ")[1:], " ")
	}

	log.Printf("🤖 Telegram command: %s from %s", command, message.From.UserName)

	switch command {
	case "/start":
		c.sendMessage(message.Chat.ID, "🌉 DCBot Bridge activated!\n\nAvailable commands:\n/help - Show help\n/status - Bridge status\n/bridge - Bridge management")

	case "/help":
		helpText := `🤖 DCBot Commands:
/start - Start the bot
/help - Show this help
/status - Show bridge status
/bridge - Bridge this chat with other platforms
/unbridge - Remove bridge connections

💡 The bot will bridge messages between Telegram and Discord platforms.`
		c.sendMessage(message.Chat.ID, helpText)

	case "/status":
		statusText := "🌉 Bridge Status:\n"
		statusText += "• Telegram: ✅ Connected\n"
		statusText += "• Discord: ⏳ Checking...\n"
		c.sendMessage(message.Chat.ID, statusText)

	case "/bridge":
		bridgeText := "🔗 Bridge Management:\n\n"
		bridgeText += "To bridge this chat with other platforms, an admin needs to configure the bridge settings.\n\n"
		bridgeText += "Current chat ID: " + strconv.FormatInt(message.Chat.ID, 10)
		c.sendMessage(message.Chat.ID, bridgeText)

	case "/unbridge":
		c.sendMessage(message.Chat.ID, "🔗 Unbridge functionality will be implemented in the next phase.")

	default:
		c.sendMessage(message.Chat.ID, "❓ Unknown command. Type /help for available commands.")
	}
}

// SendMessage sends a text message to a Telegram chat
func (c *Client) SendMessage(chatID, message string) error {
	// Parse chat ID
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %v", err)
	}

	return c.sendMessage(id, message)
}

// sendMessage internal method to send message
func (c *Client) sendMessage(chatID int64, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = tgbotapi.ModeMarkdown

	_, err := c.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send Telegram message: %v", err)
	}

	log.Printf("✅ Message sent to Telegram chat %d", chatID)
	return nil
}

// SendReply sends a reply to a specific message
func (c *Client) SendReply(chatID, replyToMessageID, message string) error {
	// Parse chat ID
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %v", err)
	}

	// Parse message ID
	msgID, err := strconv.Atoi(replyToMessageID)
	if err != nil {
		return fmt.Errorf("invalid message ID: %v", err)
	}

	msg := tgbotapi.NewMessage(id, message)
	msg.ReplyToMessageID = msgID
	msg.ParseMode = tgbotapi.ModeMarkdown

	_, err = c.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send Telegram reply: %v", err)
	}

	log.Printf("✅ Reply sent to Telegram chat %d", id)
	return nil
}

// Stop stops the Telegram bot
func (c *Client) Stop() error {
	if !c.isRunning {
		return nil
	}

	log.Println("🛑 Stopping Telegram bot...")
	
	// Stop the updates channel
	c.bot.StopReceivingUpdates()
	
	c.isRunning = false
	close(c.stopChan)
	
	log.Println("✅ Telegram bot stopped")
	return nil
}

// IsRunning returns whether the client is currently running
func (c *Client) IsRunning() bool {
	return c.isRunning
}

// GetChatInfo returns information about the configured chat
func (c *Client) GetChatInfo() (*tgbotapi.Chat, error) {
	chatConfig := tgbotapi.ChatInfoConfig{
		ChatConfig: tgbotapi.ChatConfig{
			ChatID: c.chatID,
		},
	}

	chat, err := c.bot.GetChat(chatConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat info: %v", err)
	}

	return &chat, nil
}

// userMappings stores user ID to display name mappings
var userMappings = make(map[string]string)

// messageHandlerCallback stores the bridge message handler
var messageHandlerCallback func(string, string, string, string, string) error

// storeUserMapping stores user mapping for consistent display names
func (c *Client) storeUserMapping(userID, username string) {
	if username != "" && userID != "" {
		userMappings[userID] = username
		log.Printf("📝 Stored Telegram user mapping: %s -> %s", userID, username)
	}
}

// getUserDisplayName gets the display name for a user ID
func (c *Client) getUserDisplayName(userID string) string {
	if displayName, exists := userMappings[userID]; exists {
		return displayName
	}
	return "User" + userID
}

// GetUserDisplayName returns the display name for a user ID (public method)
func (c *Client) GetUserDisplayName(userID string) string {
	return c.getUserDisplayName(userID)
}
