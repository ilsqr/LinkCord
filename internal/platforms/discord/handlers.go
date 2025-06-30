package discord

import (
	"fmt"
	"log"
	"strings"
	"time"

	"dcbot/internal/types"
	"github.com/bwmarrin/discordgo"
)

// MessageHandler handles Discord events and admin commands
type MessageHandler struct {
	client             *Client
	bridgeFunc         func(platform, channelID, userID, messageType, content string) error
	adminUsers         []string                                               // Discord user IDs
	adminRoles         []string                                               // Discord role IDs that have admin permissions
	bridgedChannels    map[string]map[string]string                          // channelID -> platform -> targetID
	bridgeCore         types.BridgeCore                                      // Bridge core interface
}

// NewMessageHandler creates a new Discord message handler
func NewMessageHandler(client *Client, bridgeFunc func(string, string, string, string, string) error) *MessageHandler {
	return &MessageHandler{
		client:          client,
		bridgeFunc:      bridgeFunc,
		adminUsers:      []string{},
		adminRoles:      []string{},
		bridgedChannels: make(map[string]map[string]string),
	}
}

// SetBridgeCore sets the bridge core reference
func (h *MessageHandler) SetBridgeCore(bc types.BridgeCore) {
	h.bridgeCore = bc
}

// SetupHandlers sets up all Discord event handlers
func (h *MessageHandler) SetupHandlers() {
	h.client.SetReadyHandler(h.onReady)
	h.client.SetMessageHandler(h.onMessageCreate)
	h.client.SetInteractionHandler(h.onInteractionCreate)
}

// onReady handles the ready event
func (h *MessageHandler) onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("🤖 Discord bot logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	
	// Register slash commands
	err := h.client.RegisterCommands()
	if err != nil {
		log.Printf("❌ Failed to register Discord commands: %v", err)
	}
}

// onMessageCreate handles new messages
func (h *MessageHandler) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Ignore webhook messages to prevent infinite loops
	if m.WebhookID != "" {
		log.Printf("⏭️ Ignoring webhook message from %s in %s", m.Author.Username, m.ChannelID)
		return
	}

	// Ignore bot messages to prevent infinite loops
	if m.Author.Bot {
		log.Printf("⏭️ Ignoring bot message from %s in %s", m.Author.Username, m.ChannelID)
		return
	}

	// Log the message
	log.Printf("🔄 Processing Discord message from %s in %s: %s", m.Author.Username, m.ChannelID, m.Content)

	// Set user mapping in bridge core for username display
	if h.bridgeCore != nil {
		username := m.Author.Username
		if username == "" {
			username = m.Author.GlobalName
		}
		if username == "" {
			username = "User" + m.Author.ID
		}
		h.bridgeCore.SetUserMapping("discord", m.Author.ID, username)
	}

	// Check if channel is bridged using bridge core first
	if h.bridgeCore != nil {
		bridges := h.bridgeCore.GetBridges(m.ChannelID)
		if len(bridges) > 0 {
			// Bridge the message using bridge core
			err := h.bridgeFunc("discord", m.ChannelID, m.Author.ID, "text", m.Content)
			if err != nil {
				log.Printf("❌ Failed to bridge Discord message: %v", err)
				h.sendErrorMessage(m.ChannelID, "Failed to bridge message to other platforms")
			}
			return
		}
	}

	// Fallback to old bridged channels method
	if bridges, exists := h.bridgedChannels[m.ChannelID]; exists && len(bridges) > 0 {
		// Bridge the message to other platforms
		if h.bridgeFunc != nil {
			err := h.bridgeFunc("discord", m.ChannelID, m.Author.ID, "text", m.Content)
			if err != nil {
				log.Printf("❌ Failed to bridge Discord message: %v", err)
				h.sendErrorMessage(m.ChannelID, "Failed to bridge message to other platforms")
			}
		}
	}
}

// onInteractionCreate handles slash command interactions
func (h *MessageHandler) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if user has admin permissions
	if !h.isAdmin(i.Member) {
		h.respondToInteraction(s, i, "❌ You don't have permission to use this command.")
		return
	}

	data := i.ApplicationCommandData()
	
	switch data.Name {
	case "bridge":
		h.handleBridgeCommand(s, i)
	case "config":
		h.handleConfigCommand(s, i)
	case "help":
		h.handleHelpCommand(s, i)
	default:
		h.respondToInteraction(s, i, "❓ Unknown command")
	}
}

// handleBridgeCommand handles bridge-related commands
func (h *MessageHandler) handleBridgeCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	
	if len(data.Options) == 0 {
		h.respondToInteraction(s, i, "❌ No subcommand specified")
		return
	}

	subcommand := data.Options[0]
	
	switch subcommand.Name {
	case "status":
		h.commandBridgeStatus(s, i)
	case "create":
		h.commandBridgeCreate(s, i, subcommand.Options)
	case "remove":
		h.commandBridgeRemove(s, i, subcommand.Options)
	default:
		h.respondToInteraction(s, i, "❓ Unknown bridge subcommand")
	}
}

// handleConfigCommand handles configuration commands
func (h *MessageHandler) handleConfigCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	
	if len(data.Options) == 0 {
		h.respondToInteraction(s, i, "❌ No subcommand specified")
		return
	}

	subcommand := data.Options[0]
	
	switch subcommand.Name {
	case "platforms":
		h.commandConfigPlatforms(s, i)
	case "channels":
		h.commandConfigChannels(s, i)
	default:
		h.respondToInteraction(s, i, "❓ Unknown config subcommand")
	}
}

// handleHelpCommand handles help command
func (h *MessageHandler) handleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "🌉 Bridge Bot Help",
		Description: "Commands to manage cross-platform bridges",
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🔗 Bridge Commands",
				Value:  "`/bridge status` - Show bridge status\n`/bridge create` - Create new bridge\n`/bridge remove` - Remove bridge",
				Inline: false,
			},
			{
				Name:   "⚙️ Config Commands",
				Value:  "`/config platforms` - Show enabled platforms\n`/config channels` - List available channels",
				Inline: false,
			},
			{
				Name:   "ℹ️ General",
				Value:  "`/help` - Show this help message",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Bridge Bot - Discord Control Center",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	h.respondToInteractionWithEmbed(s, i, embed)
}

// commandBridgeStatus shows current bridge status
func (h *MessageHandler) commandBridgeStatus(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID
	
	embed := &discordgo.MessageEmbed{
		Title: "🔗 Bridge Status",
		Color: 0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📍 Current Channel",
				Value:  fmt.Sprintf("<#%s>", channelID),
				Inline: true,
			},
		},
	}

	// Check if current channel has bridges using bridge core
	bridgeList := ""
	if h.bridgeCore != nil {
		bridges := h.bridgeCore.GetBridges(channelID)
		if len(bridges) > 0 {
			for _, bridge := range bridges {
				bridgeList += fmt.Sprintf("• **%s**: `%s`\n", strings.Title(bridge.TargetPlatform), bridge.TargetChannelID)
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🌉 Active Bridges",
				Value:  bridgeList,
				Inline: false,
			})
			embed.Color = 0x00ff00
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🌉 Active Bridges",
				Value:  "No bridges configured for this channel",
				Inline: false,
			})
		}

		// Add platform status using bridge core
		platformStatus := ""
		if statusMap := h.bridgeCore.GetPlatformStatus(); len(statusMap) > 0 {
			for platform, isConnected := range statusMap {
				status := "❌ Disconnected"
				if isConnected {
					status = "✅ Connected"
				}
				platformStatus += fmt.Sprintf("• **%s**: %s\n", strings.Title(platform), status)
			}
		} else {
			platformStatus = "• **Discord**: ✅ Active (Control Center)\n• **Telegram**: ⏳ Checking..."
		}
		
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🔌 Platform Status",
			Value:  platformStatus,
			Inline: false,
		})
	} else {
		// Fallback to old method
		if bridges, exists := h.bridgedChannels[channelID]; exists && len(bridges) > 0 {
			for platform, targetID := range bridges {
				bridgeList += fmt.Sprintf("• **%s**: `%s`\n", strings.Title(platform), targetID)
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🌉 Active Bridges",
				Value:  bridgeList,
				Inline: false,
			})
			embed.Color = 0x00ff00
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🌉 Active Bridges",
				Value:  "No bridges configured for this channel",
				Inline: false,
			})
		}

		// Add platform status
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🔌 Platform Status",
			Value:  "• **Discord**: ✅ Active (Control Center)\n• **Telegram**: ⏳ Checking...",
			Inline: false,
		})
	}

	h.respondToInteractionWithEmbed(s, i, embed)
}

// commandBridgeCreate creates a new bridge
func (h *MessageHandler) commandBridgeCreate(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) < 2 {
		h.respondToInteraction(s, i, "❌ Missing required parameters")
		return
	}

	platform := options[0].StringValue()
	targetRoom := options[1].StringValue()
	channelID := i.ChannelID

	// Use bridge core if available
	if h.bridgeCore != nil {
		err := h.bridgeCore.AddBridge("discord", channelID, platform, targetRoom)
		if err != nil {
			h.respondToInteraction(s, i, fmt.Sprintf("❌ Failed to create bridge: %v", err))
			return
		}
	} else {
		// Fallback to old method
		if h.bridgedChannels[channelID] == nil {
			h.bridgedChannels[channelID] = make(map[string]string)
		}
		h.bridgedChannels[channelID][platform] = targetRoom
	}

	embed := &discordgo.MessageEmbed{
		Title: "✅ Bridge Created",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Discord Channel",
				Value:  fmt.Sprintf("<#%s>", channelID),
				Inline: true,
			},
			{
				Name:   "Target Platform",
				Value:  strings.Title(platform),
				Inline: true,
			},
			{
				Name:   "Target Room",
				Value:  fmt.Sprintf("`%s`", targetRoom),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Bridge is now active - messages will be synchronized",
		},
	}

	h.respondToInteractionWithEmbed(s, i, embed)
	log.Printf("🌉 Bridge created: Discord %s ↔ %s %s", channelID, platform, targetRoom)
}

// commandBridgeRemove removes a bridge
func (h *MessageHandler) commandBridgeRemove(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) < 1 {
		h.respondToInteraction(s, i, "❌ Missing platform parameter")
		return
	}

	platform := options[0].StringValue()
	channelID := i.ChannelID

	// Use bridge core if available
	if h.bridgeCore != nil {
		err := h.bridgeCore.RemoveBridge(channelID, platform)
		if err != nil {
			h.respondToInteraction(s, i, fmt.Sprintf("❌ Failed to remove bridge: %v", err))
			return
		}
	} else {
		// Fallback to old method
		if h.bridgedChannels[channelID] == nil {
			h.respondToInteraction(s, i, "❌ No bridges configured for this channel")
			return
		}

		if _, exists := h.bridgedChannels[channelID][platform]; !exists {
			h.respondToInteraction(s, i, fmt.Sprintf("❌ No %s bridge found for this channel", platform))
			return
		}

		// Remove the bridge
		delete(h.bridgedChannels[channelID], platform)

		// Clean up empty channel entry
		if len(h.bridgedChannels[channelID]) == 0 {
			delete(h.bridgedChannels, channelID)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: "🗑️ Bridge Removed",
		Color: 0xff9900,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Platform",
				Value:  strings.Title(platform),
				Inline: true,
			},
			{
				Name:   "Channel",
				Value:  fmt.Sprintf("<#%s>", channelID),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Bridge removed - messages will no longer be synchronized",
		},
	}

	h.respondToInteractionWithEmbed(s, i, embed)
	log.Printf("🗑️ Bridge removed: %s bridge for Discord channel %s", platform, channelID)
}

// commandConfigPlatforms shows enabled platforms
func (h *MessageHandler) commandConfigPlatforms(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title: "🔌 Platform Configuration",
		Color: 0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Available Platforms",
				Value:  "• **Discord**: ✅ Active (Control Center)\n• **Telegram**: ⏳ Checking connection...",
				Inline: false,
			},
			{
				Name:   "Bridge Capabilities",
				Value:  "• Text messages\n• User identification\n• Bidirectional sync",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /bridge create to establish connections",
		},
	}

	h.respondToInteractionWithEmbed(s, i, embed)
}

// commandConfigChannels lists available channels
func (h *MessageHandler) commandConfigChannels(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channels, err := h.client.GetGuildChannels()
	if err != nil {
		h.respondToInteraction(s, i, "❌ Failed to get channel list")
		return
	}

	textChannels := ""
	bridgedCount := 0

	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildText {
			bridgeStatus := ""
			if bridges, exists := h.bridgedChannels[channel.ID]; exists && len(bridges) > 0 {
				bridgeStatus = " 🌉"
				bridgedCount++
			}
			textChannels += fmt.Sprintf("• <#%s>%s\n", channel.ID, bridgeStatus)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: "📋 Channel Configuration",
		Color: 0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Text Channels",
				Value:  textChannels,
				Inline: false,
			},
			{
				Name:   "Statistics",
				Value:  fmt.Sprintf("🌉 Bridged Channels: %d", bridgedCount),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "🌉 = Has active bridges",
		},
	}

	h.respondToInteractionWithEmbed(s, i, embed)
}

// isAdmin checks if a member has admin permissions
func (h *MessageHandler) isAdmin(member *discordgo.Member) bool {
	if member == nil {
		return false
	}

	// Check if user is in admin users list
	for _, adminID := range h.adminUsers {
		if member.User.ID == adminID {
			return true
		}
	}

	// Check if user has any admin roles
	for _, userRole := range member.Roles {
		for _, adminRole := range h.adminRoles {
			if userRole == adminRole {
				return true
			}
		}
	}

	// For development, allow users with administrative permissions
	// TODO: Remove this in production and rely on configured admin lists
	if member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}

	return false
}

// sendErrorMessage sends an error message to a channel
func (h *MessageHandler) sendErrorMessage(channelID, errorMsg string) {
	message := fmt.Sprintf("❌ Error: %s", errorMsg)
	err := h.client.SendMessage(channelID, message)
	if err != nil {
		log.Printf("❌ Failed to send error message: %v", err)
	}
}

// respondToInteraction sends a response to a slash command interaction
func (h *MessageHandler) respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
	if err != nil {
		log.Printf("❌ Failed to respond to interaction: %v", err)
	}
}

// respondToInteractionWithEmbed sends an embed response to a slash command interaction
func (h *MessageHandler) respondToInteractionWithEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("❌ Failed to respond to interaction with embed: %v", err)
	}
}

// SetAdminUsers sets the list of admin user IDs
func (h *MessageHandler) SetAdminUsers(adminUsers []string) {
	h.adminUsers = adminUsers
	log.Printf("👮 Discord admin users updated: %v", adminUsers)
}

// SetAdminRoles sets the list of admin role IDs
func (h *MessageHandler) SetAdminRoles(adminRoles []string) {
	h.adminRoles = adminRoles
	log.Printf("👮 Discord admin roles updated: %v", adminRoles)
}

// GetBridgedChannels returns the current bridge configuration
func (h *MessageHandler) GetBridgedChannels() map[string]map[string]string {
	return h.bridgedChannels
}

// AddBridge adds a bridge programmatically
func (h *MessageHandler) AddBridge(channelID, platform, targetID string) {
	if h.bridgedChannels[channelID] == nil {
		h.bridgedChannels[channelID] = make(map[string]string)
	}
	h.bridgedChannels[channelID][platform] = targetID
	log.Printf("🌉 Bridge added: Discord %s ↔ %s %s", channelID, platform, targetID)
}

// RemoveBridge removes a bridge programmatically
func (h *MessageHandler) RemoveBridge(channelID, platform string) {
	if h.bridgedChannels[channelID] != nil {
		delete(h.bridgedChannels[channelID], platform)
		if len(h.bridgedChannels[channelID]) == 0 {
			delete(h.bridgedChannels, channelID)
		}
		log.Printf("🗑️ Bridge removed: %s bridge for Discord channel %s", platform, channelID)
	}
}
