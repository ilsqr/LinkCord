package telegram

import (
	"log"
	"strconv"
	"strings"
)

// MessageHandler handles incoming Telegram messages and bridges them to other platforms
type MessageHandler struct {
	client      *Client
	bridgeFunc  func(platform, chatID, userID, messageType, content string) error
	allowedChats []int64
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(client *Client, bridgeFunc func(string, string, string, string, string) error) *MessageHandler {
	return &MessageHandler{
		client:       client,
		bridgeFunc:   bridgeFunc,
		allowedChats: []int64{}, // Will be configured later
	}
}

// HandleMessage processes incoming Telegram messages
func (h *MessageHandler) HandleMessage(platform, chatID, userID, messageType, content string) error {
	// Log the message
	log.Printf("🔄 Processing Telegram message from %s in %s: %s", userID, chatID, content)

	// Parse chat ID
	chatIDInt, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return nil // Skip invalid chat IDs
	}

	// Check if chat is allowed (if allowedChats is configured)
	if len(h.allowedChats) > 0 && !h.isChatAllowed(chatIDInt) {
		log.Printf("⚠️ Message from unauthorized chat: %d", chatIDInt)
		return nil
	}

	// Skip Telegram bot commands (they will be handled by Discord)
	if strings.HasPrefix(content, "/") {
		log.Printf("⏭️ Skipping Telegram command (handled by Discord): %s", content)
		return nil
	}

	// Bridge the message to other platforms
	if h.bridgeFunc != nil {
		err := h.bridgeFunc("telegram", chatID, userID, messageType, content)
		if err != nil {
			log.Printf("❌ Failed to bridge Telegram message: %v", err)
			return err
		}
	}

	return nil
}

// isChatAllowed checks if chat is allowed to use the bridge
func (h *MessageHandler) isChatAllowed(chatID int64) bool {
	// If no allowed chats configured, allow all
	if len(h.allowedChats) == 0 {
		return true
	}

	for _, allowedID := range h.allowedChats {
		if allowedID == chatID {
			return true
		}
	}
	return false
}

// SetAllowedChats sets the list of allowed chats
func (h *MessageHandler) SetAllowedChats(allowedChats []int64) {
	h.allowedChats = allowedChats
	log.Printf("💬 Telegram allowed chats updated: %v", allowedChats)
}

// AddAllowedChat adds a chat to the allowed list
func (h *MessageHandler) AddAllowedChat(chatID int64) {
	// Check if already exists
	for _, existing := range h.allowedChats {
		if existing == chatID {
			return
		}
	}
	
	h.allowedChats = append(h.allowedChats, chatID)
	log.Printf("💬 Added allowed chat: %d", chatID)
}

// RemoveAllowedChat removes a chat from the allowed list
func (h *MessageHandler) RemoveAllowedChat(chatID int64) {
	for i, existing := range h.allowedChats {
		if existing == chatID {
			h.allowedChats = append(h.allowedChats[:i], h.allowedChats[i+1:]...)
			log.Printf("💬 Removed allowed chat: %d", chatID)
			return
		}
	}
}
