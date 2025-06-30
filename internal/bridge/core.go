package bridge

import (
	"fmt"
	"log"
	"time"

	"dcbot/internal/database"
	"dcbot/internal/database/models"
	"dcbot/internal/types"
)

// BridgeCore manages message bridging between platforms
type BridgeCore struct {
	platforms    map[string]types.Platform
	connections  map[string][]*types.BridgeConnection // sourceChannelID -> connections
	userMappings map[string]map[string]string         // platform -> userID -> displayName
	db           *database.Database                   // Database for persistence
}

// NewBridgeCore creates a new bridge core instance
func NewBridgeCore(db *database.Database) *BridgeCore {
	bc := &BridgeCore{
		platforms:    make(map[string]types.Platform),
		connections:  make(map[string][]*types.BridgeConnection),
		userMappings: make(map[string]map[string]string),
		db:           db,
	}
	
	// Load existing bridges from database
	if err := bc.loadBridgesFromDB(); err != nil {
		log.Printf("⚠️ Failed to load bridges from database: %v", err)
	}
	
	return bc
}

// loadBridgesFromDB loads existing bridge configurations from database
func (bc *BridgeCore) loadBridgesFromDB() error {
	if bc.db == nil {
		return fmt.Errorf("database not initialized")
	}

	bridges, err := bc.db.GetAllActiveBridges()
	if err != nil {
		return fmt.Errorf("failed to get active bridges: %v", err)
	}

	bridgeCount := 0
	roomGroups := make(map[int][]*models.RoomMapping) // room_id -> mappings

	// Group mappings by room_id
	for _, mappings := range bridges {
		for _, mapping := range mappings {
			roomGroups[mapping.RoomID] = append(roomGroups[mapping.RoomID], mapping)
		}
	}

	// Create bridge connections for each room with multiple platforms
	for roomID, mappings := range roomGroups {
		if len(mappings) < 2 {
			continue // Need at least 2 platforms for a bridge
		}

		// Create bidirectional connections between all platforms in this room
		for i, source := range mappings {
			for j, target := range mappings {
				if i == j {
					continue // Skip self-connection
				}

				connection := &types.BridgeConnection{
					ID:              fmt.Sprintf("db_%d_%s_%s_%s_%s", roomID, source.Platform, source.PlatformRoomID, target.Platform, target.PlatformRoomID),
					SourcePlatform:  source.Platform,
					SourceChannelID: source.PlatformRoomID,
					TargetPlatform:  target.Platform,
					TargetChannelID: target.PlatformRoomID,
					IsActive:        true,
					CreatedAt:       source.CreatedAt,
				}

				// Add to connections map
				if bc.connections[source.PlatformRoomID] == nil {
					bc.connections[source.PlatformRoomID] = make([]*types.BridgeConnection, 0)
				}
				bc.connections[source.PlatformRoomID] = append(bc.connections[source.PlatformRoomID], connection)
				bridgeCount++
			}
		}
	}

	if bridgeCount > 0 {
		log.Printf("✅ Loaded %d bridge connections from database", bridgeCount)
	}
	return nil
}

// RegisterPlatform registers a platform with the bridge core
func (bc *BridgeCore) RegisterPlatform(platform types.Platform) {
	bc.platforms[platform.GetName()] = platform
	if bc.userMappings[platform.GetName()] == nil {
		bc.userMappings[platform.GetName()] = make(map[string]string)
	}
	log.Printf("🔌 Platform registered: %s", platform.GetName())
}

// AddBridge creates a new bridge connection and persists it to database
func (bc *BridgeCore) AddBridge(sourcePlatform, sourceChannelID, targetPlatform, targetChannelID string) error {
	// Validate platforms
	if _, exists := bc.platforms[sourcePlatform]; !exists {
		return fmt.Errorf("source platform %s not registered", sourcePlatform)
	}
	if _, exists := bc.platforms[targetPlatform]; !exists {
		return fmt.Errorf("target platform %s not registered", targetPlatform)
	}

	// Persist to database if available
	if bc.db != nil {
		if err := bc.saveBridgeToDatabase(sourcePlatform, sourceChannelID, targetPlatform, targetChannelID); err != nil {
			return fmt.Errorf("failed to save bridge to database: %v", err)
		}
	}

	// Create bridge connections in memory
	connection := &types.BridgeConnection{
		ID:              fmt.Sprintf("%s_%s_%s_%s", sourcePlatform, sourceChannelID, targetPlatform, targetChannelID),
		SourcePlatform:  sourcePlatform,
		SourceChannelID: sourceChannelID,
		TargetPlatform:  targetPlatform,
		TargetChannelID: targetChannelID,
		IsActive:        true,
		CreatedAt:       time.Now(),
	}

	// Add to connections map
	if bc.connections[sourceChannelID] == nil {
		bc.connections[sourceChannelID] = make([]*types.BridgeConnection, 0)
	}
	bc.connections[sourceChannelID] = append(bc.connections[sourceChannelID], connection)

	// Also add reverse connection for bidirectional bridging
	reverseConnection := &types.BridgeConnection{
		ID:              fmt.Sprintf("%s_%s_%s_%s", targetPlatform, targetChannelID, sourcePlatform, sourceChannelID),
		SourcePlatform:  targetPlatform,
		SourceChannelID: targetChannelID,
		TargetPlatform:  sourcePlatform,
		TargetChannelID: sourceChannelID,
		IsActive:        true,
		CreatedAt:       time.Now(),
	}

	if bc.connections[targetChannelID] == nil {
		bc.connections[targetChannelID] = make([]*types.BridgeConnection, 0)
	}
	bc.connections[targetChannelID] = append(bc.connections[targetChannelID], reverseConnection)

	log.Printf("🌉 Bridge added: %s #%s ↔ %s #%s", sourcePlatform, sourceChannelID, targetPlatform, targetChannelID)
	return nil
}

// saveBridgeToDatabase saves a bridge configuration to the database
func (bc *BridgeCore) saveBridgeToDatabase(sourcePlatform, sourceChannelID, targetPlatform, targetChannelID string) error {
	// Create a unique room name for this bridge
	roomName := fmt.Sprintf("bridge_%s_%s_%s_%s", sourcePlatform, sourceChannelID, targetPlatform, targetChannelID)
	
	// Create or get room
	room, err := bc.db.CreateOrGetRoom(roomName)
	if err != nil {
		return fmt.Errorf("failed to create/get room: %v", err)
	}

	// Create room mappings for both platforms
	_, err = bc.db.CreateOrGetRoomMapping(room.ID, sourcePlatform, sourceChannelID, 
		fmt.Sprintf("%s_%s", sourcePlatform, sourceChannelID), "channel")
	if err != nil {
		return fmt.Errorf("failed to create source room mapping: %v", err)
	}

	_, err = bc.db.CreateOrGetRoomMapping(room.ID, targetPlatform, targetChannelID,
		fmt.Sprintf("%s_%s", targetPlatform, targetChannelID), "channel")
	if err != nil {
		return fmt.Errorf("failed to create target room mapping: %v", err)
	}

	// Create bridge config
	_, err = bc.db.CreateOrGetBridgeConfig(room.ID)
	if err != nil {
		return fmt.Errorf("failed to create bridge config: %v", err)
	}

	return nil
}

// RemoveBridge removes a bridge connection and updates database
func (bc *BridgeCore) RemoveBridge(sourceChannelID, targetPlatform string) error {
	connections := bc.connections[sourceChannelID]
	if connections == nil {
		return fmt.Errorf("no bridges found for channel %s", sourceChannelID)
	}

	var removedConnection *types.BridgeConnection

	// Find and remove the connection
	for i, conn := range connections {
		if conn.TargetPlatform == targetPlatform {
			removedConnection = conn
			// Remove from source connections
			bc.connections[sourceChannelID] = append(connections[:i], connections[i+1:]...)
			
			// Remove reverse connection
			reverseConnections := bc.connections[conn.TargetChannelID]
			for j, reverseConn := range reverseConnections {
				if reverseConn.TargetChannelID == sourceChannelID && reverseConn.TargetPlatform == conn.SourcePlatform {
					bc.connections[conn.TargetChannelID] = append(reverseConnections[:j], reverseConnections[j+1:]...)
					break
				}
			}
			break
		}
	}

	if removedConnection == nil {
		return fmt.Errorf("bridge to %s not found for channel %s", targetPlatform, sourceChannelID)
	}

	// Remove from database if available
	if bc.db != nil {
		if err := bc.removeBridgeFromDatabase(removedConnection.SourcePlatform, sourceChannelID, targetPlatform); err != nil {
			log.Printf("⚠️ Failed to remove bridge from database: %v", err)
		}
	}

	log.Printf("🗑️ Bridge removed: %s #%s ↔ %s #%s", removedConnection.SourcePlatform, sourceChannelID, targetPlatform, removedConnection.TargetChannelID)
	return nil
}

// removeBridgeFromDatabase removes a bridge from the database
func (bc *BridgeCore) removeBridgeFromDatabase(sourcePlatform, sourceChannelID, targetPlatform string) error {
	// Find the room mapping for source channel
	sourceMapping, err := bc.db.GetRoomMappingByPlatformRoom(sourcePlatform, sourceChannelID)
	if err != nil {
		return fmt.Errorf("source room mapping not found: %v", err)
	}

	// Remove the target platform mapping from this room
	err = bc.db.RemoveRoomMapping(sourceMapping.RoomID, targetPlatform)
	if err != nil {
		return fmt.Errorf("failed to remove target room mapping: %v", err)
	}

	return nil
}


// ProcessMessage processes and bridges a message to connected platforms
func (bc *BridgeCore) ProcessMessage(message *types.BridgeMessage) error {
	// Get connections for this channel
	connections := bc.connections[message.SourceChannelID]
	if len(connections) == 0 {
		log.Printf("⚠️ No bridges configured for %s channel %s", message.SourcePlatform, message.SourceChannelID)
		return nil
	}

	log.Printf("🔄 Processing message from %s (room: %s): %s", message.SourcePlatform, message.SourceChannelID, message.Content)
	log.Printf("   Found %d bridge connections for this channel", len(connections))

	// Bridge to all connected platforms
	for _, connection := range connections {
		if !connection.IsActive {
			log.Printf("⏭️ Skipping inactive bridge: %s → %s", connection.SourcePlatform, connection.TargetPlatform)
			continue
		}

		log.Printf("🎯 Attempting to bridge message: %s → %s (channel: %s)", 
			connection.SourcePlatform, connection.TargetPlatform, connection.TargetChannelID)

		targetPlatform := bc.platforms[connection.TargetPlatform]
		if targetPlatform == nil || !targetPlatform.IsConnected() {
			log.Printf("⚠️ Target platform %s not available or not connected", connection.TargetPlatform)
			continue
		}

		// Special handling for Discord webhook messages
		if connection.TargetPlatform == types.PlatformDiscord {
			if discordAdapter, ok := targetPlatform.(*DiscordAdapter); ok {
				err := discordAdapter.SendBridgeMessage(connection.TargetChannelID, message)
				if err != nil {
					log.Printf("❌ Failed to send Discord webhook message: %v", err)
					// Fallback to regular message
					formattedMessage := targetPlatform.FormatMessage(message)
					err = targetPlatform.SendMessage(connection.TargetChannelID, formattedMessage)
				}
			} else {
				// Fallback to regular message
				formattedMessage := targetPlatform.FormatMessage(message)
				err := targetPlatform.SendMessage(connection.TargetChannelID, formattedMessage)
				if err != nil {
					log.Printf("❌ Failed to bridge message to %s: %v", connection.TargetPlatform, err)
					continue
				}
			}
		} else {
			// Send regular message
			formattedMessage := targetPlatform.FormatMessage(message)
			err := targetPlatform.SendMessage(connection.TargetChannelID, formattedMessage)
			if err != nil {
				log.Printf("❌ Failed to bridge message to %s: %v", connection.TargetPlatform, err)
				continue
			}
		}

		log.Printf("✅ Message bridged: %s → %s", message.SourcePlatform, connection.TargetPlatform)
	}

	return nil
}

// ProcessMessageLegacy processes and bridges a message (legacy method for backward compatibility)
func (bc *BridgeCore) ProcessMessageLegacy(sourcePlatform, channelID, userID, messageType, content string) error {
	log.Printf("🔄 ProcessMessageLegacy called:")
	log.Printf("   Platform: %s", sourcePlatform)
	log.Printf("   Channel: %s", channelID)
	log.Printf("   User: %s", userID)
	log.Printf("   Type: %s", messageType)
	log.Printf("   Content: %s", content)
	
	// Check if we have any connections for this channel
	connections := bc.connections[channelID]
	log.Printf("   Connections found: %d", len(connections))
	
	if len(connections) == 0 {
		log.Printf("   ⚠️ No bridge connections found for channel %s", channelID)
		return nil
	}

	// Create bridge message
	message := &types.BridgeMessage{
		ID:              fmt.Sprintf("%s_%s_%d", sourcePlatform, channelID, time.Now().Unix()),
		SourcePlatform:  sourcePlatform,
		SourceChannelID: channelID,
		SourceUserID:    userID,
		Username:        bc.getDisplayName(sourcePlatform, userID),
		Content:         content,
		MessageType:     messageType,
		Timestamp:       time.Now(),
	}

	return bc.ProcessMessage(message)
}

// GetBridges returns all bridge connections for a channel
func (bc *BridgeCore) GetBridges(channelID string) []*types.BridgeConnection {
	return bc.connections[channelID]
}

// GetAllBridges returns all bridge connections
func (bc *BridgeCore) GetAllBridges() map[string][]*types.BridgeConnection {
	return bc.connections
}

// SetUserMapping sets a display name for a user on a platform
func (bc *BridgeCore) SetUserMapping(platform, userID, displayName string) {
	if bc.userMappings[platform] == nil {
		bc.userMappings[platform] = make(map[string]string)
	}
	bc.userMappings[platform][userID] = displayName
}

// getDisplayName gets the display name for a user, falling back to user ID
func (bc *BridgeCore) getDisplayName(platform, userID string) string {
	if bc.userMappings[platform] != nil {
		if displayName, exists := bc.userMappings[platform][userID]; exists {
			return displayName
		}
	}
	return userID
}

// GetPlatformStatus returns the status of all registered platforms
func (bc *BridgeCore) GetPlatformStatus() map[string]bool {
	status := make(map[string]bool)
	for name, platform := range bc.platforms {
		status[name] = platform.IsConnected()
	}
	return status
}

// GetBridgeStats returns statistics about the bridge system
func (bc *BridgeCore) GetBridgeStats() map[string]int {
	stats := make(map[string]int)
	
	totalBridges := 0
	activeBridges := 0
	
	for _, connections := range bc.connections {
		for _, conn := range connections {
			totalBridges++
			if conn.IsActive {
				activeBridges++
			}
		}
	}
	
	// Divide by 2 because we count bidirectional bridges twice
	stats["total_bridges"] = totalBridges / 2
	stats["active_bridges"] = activeBridges / 2
	stats["registered_platforms"] = len(bc.platforms)
	stats["bridged_channels"] = len(bc.connections)
	
	return stats
}


