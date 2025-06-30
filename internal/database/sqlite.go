package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"dcbot/internal/database/models"
	_ "modernc.org/sqlite"
)

type Database struct {
	db *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(dbPath string) (*Database, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	database := &Database{db: db}

	// Run migrations
	if err := database.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %v", err)
	}

	log.Println("✅ Database connected and migrated successfully")
	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// GetDB returns the underlying sql.DB instance
func (d *Database) GetDB() *sql.DB {
	return d.db
}

// migrate runs database migrations
func (d *Database) migrate() error {
	migrations := []string{
		createUsersTable,
		createUserMappingsTable,
		createRoomsTable,
		createRoomMappingsTable,
		createMessagesTable,
		createMessageMappingsTable,
		createBridgeConfigTable,
		createIndexes,
	}

	for i, migration := range migrations {
		if _, err := d.db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %v", i+1, err)
		}
	}

	log.Println("✅ Database migrations completed")
	return nil
}

// Migration SQL statements
const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

const createUserMappingsTable = `
CREATE TABLE IF NOT EXISTS user_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    platform TEXT NOT NULL,
    platform_user_id TEXT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(platform, platform_user_id)
);`

const createRoomsTable = `
CREATE TABLE IF NOT EXISTS rooms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

const createRoomMappingsTable = `
CREATE TABLE IF NOT EXISTS room_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id INTEGER NOT NULL,
    platform TEXT NOT NULL,
    platform_room_id TEXT NOT NULL,
    room_name TEXT NOT NULL DEFAULT '',
    room_type TEXT NOT NULL DEFAULT 'channel',
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    UNIQUE(platform, platform_room_id)
);`

const createMessagesTable = `
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    original_id TEXT NOT NULL,
    source_platform TEXT NOT NULL,
    source_room_id TEXT NOT NULL,
    source_user_id TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    message_type TEXT NOT NULL DEFAULT 'text',
    media_url TEXT NOT NULL DEFAULT '',
    media_mime_type TEXT NOT NULL DEFAULT '',
    reply_to_id INTEGER,
    is_edited BOOLEAN NOT NULL DEFAULT 0,
    is_deleted BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (reply_to_id) REFERENCES messages(id),
    UNIQUE(source_platform, original_id)
);`

const createMessageMappingsTable = `
CREATE TABLE IF NOT EXISTS message_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id INTEGER NOT NULL,
    platform TEXT NOT NULL,
    platform_msg_id TEXT NOT NULL,
    platform_room_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    UNIQUE(platform, platform_msg_id)
);`

const createBridgeConfigTable = `
CREATE TABLE IF NOT EXISTS bridge_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    allow_media BOOLEAN NOT NULL DEFAULT 1,
    allow_edits BOOLEAN NOT NULL DEFAULT 1,
    allow_deletes BOOLEAN NOT NULL DEFAULT 1,
    filter_words TEXT NOT NULL DEFAULT '[]',
    max_message_length INTEGER NOT NULL DEFAULT 4000,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    UNIQUE(room_id)
);`

const createIndexes = `
CREATE INDEX IF NOT EXISTS idx_user_mappings_platform_user_id ON user_mappings(platform, platform_user_id);
CREATE INDEX IF NOT EXISTS idx_room_mappings_platform_room_id ON room_mappings(platform, platform_room_id);
CREATE INDEX IF NOT EXISTS idx_messages_source ON messages(source_platform, source_room_id);
CREATE INDEX IF NOT EXISTS idx_message_mappings_platform ON message_mappings(platform, platform_msg_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);
`

// Bridge persistence methods

// CreateOrGetRoom creates a room if it doesn't exist, or returns existing room
func (d *Database) CreateOrGetRoom(name string) (*models.Room, error) {
	// First try to get existing room
	var room models.Room
	err := d.db.QueryRow("SELECT id, name, created_at, updated_at FROM rooms WHERE name = ?", name).
		Scan(&room.ID, &room.Name, &room.CreatedAt, &room.UpdatedAt)
	
	if err == nil {
		return &room, nil
	}
	
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query room: %v", err)
	}

	// Create new room
	result, err := d.db.Exec("INSERT INTO rooms (name, created_at, updated_at) VALUES (?, ?, ?)",
		name, time.Now(), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get room ID: %v", err)
	}

	room = models.Room{
		ID:        int(id),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return &room, nil
}

// CreateOrGetRoomMapping creates or updates a room mapping
func (d *Database) CreateOrGetRoomMapping(roomID int, platform, platformRoomID, roomName, roomType string) (*models.RoomMapping, error) {
	// First try to get existing mapping
	var mapping models.RoomMapping
	err := d.db.QueryRow(`
		SELECT id, room_id, platform, platform_room_id, room_name, room_type, is_active, created_at, updated_at 
		FROM room_mappings 
		WHERE room_id = ? AND platform = ? AND platform_room_id = ?`,
		roomID, platform, platformRoomID).
		Scan(&mapping.ID, &mapping.RoomID, &mapping.Platform, &mapping.PlatformRoomID, 
			&mapping.RoomName, &mapping.RoomType, &mapping.IsActive, &mapping.CreatedAt, &mapping.UpdatedAt)
	
	if err == nil {
		// Update existing mapping if needed
		_, err = d.db.Exec(`
			UPDATE room_mappings 
			SET room_name = ?, room_type = ?, is_active = 1, updated_at = ? 
			WHERE id = ?`,
			roomName, roomType, time.Now(), mapping.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to update room mapping: %v", err)
		}
		mapping.RoomName = roomName
		mapping.RoomType = roomType
		mapping.IsActive = true
		mapping.UpdatedAt = time.Now()
		return &mapping, nil
	}
	
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query room mapping: %v", err)
	}

	// Create new mapping
	result, err := d.db.Exec(`
		INSERT INTO room_mappings (room_id, platform, platform_room_id, room_name, room_type, is_active, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, 1, ?, ?)`,
		roomID, platform, platformRoomID, roomName, roomType, time.Now(), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create room mapping: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get room mapping ID: %v", err)
	}

	mapping = models.RoomMapping{
		ID:             int(id),
		RoomID:         roomID,
		Platform:       platform,
		PlatformRoomID: platformRoomID,
		RoomName:       roomName,
		RoomType:       roomType,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return &mapping, nil
}

// GetActiveRoomMappings returns all active room mappings for a room
func (d *Database) GetActiveRoomMappings(roomID int) ([]*models.RoomMapping, error) {
	rows, err := d.db.Query(`
		SELECT id, room_id, platform, platform_room_id, room_name, room_type, is_active, created_at, updated_at 
		FROM room_mappings 
		WHERE room_id = ? AND is_active = 1`,
		roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to query room mappings: %v", err)
	}
	defer rows.Close()

	var mappings []*models.RoomMapping
	for rows.Next() {
		var mapping models.RoomMapping
		err := rows.Scan(&mapping.ID, &mapping.RoomID, &mapping.Platform, &mapping.PlatformRoomID,
			&mapping.RoomName, &mapping.RoomType, &mapping.IsActive, &mapping.CreatedAt, &mapping.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan room mapping: %v", err)
		}
		mappings = append(mappings, &mapping)
	}

	return mappings, nil
}

// GetRoomMappingByPlatformRoom finds a room mapping by platform and platform room ID
func (d *Database) GetRoomMappingByPlatformRoom(platform, platformRoomID string) (*models.RoomMapping, error) {
	var mapping models.RoomMapping
	err := d.db.QueryRow(`
		SELECT id, room_id, platform, platform_room_id, room_name, room_type, is_active, created_at, updated_at 
		FROM room_mappings 
		WHERE platform = ? AND platform_room_id = ? AND is_active = 1`,
		platform, platformRoomID).
		Scan(&mapping.ID, &mapping.RoomID, &mapping.Platform, &mapping.PlatformRoomID,
			&mapping.RoomName, &mapping.RoomType, &mapping.IsActive, &mapping.CreatedAt, &mapping.UpdatedAt)
	
	if err != nil {
		return nil, err
	}

	return &mapping, nil
}

// RemoveRoomMapping deactivates a room mapping
func (d *Database) RemoveRoomMapping(roomID int, platform string) error {
	_, err := d.db.Exec(`
		UPDATE room_mappings 
		SET is_active = 0, updated_at = ? 
		WHERE room_id = ? AND platform = ?`,
		time.Now(), roomID, platform)
	return err
}

// CreateOrGetBridgeConfig creates or gets bridge configuration for a room
func (d *Database) CreateOrGetBridgeConfig(roomID int) (*models.BridgeConfig, error) {
	// First try to get existing config
	var config models.BridgeConfig
	err := d.db.QueryRow(`
		SELECT id, room_id, is_active, allow_media, allow_edits, allow_deletes, filter_words, max_message_length, created_at, updated_at 
		FROM bridge_config 
		WHERE room_id = ?`,
		roomID).
		Scan(&config.ID, &config.RoomID, &config.IsActive, &config.AllowMedia, &config.AllowEdits,
			&config.AllowDeletes, &config.FilterWords, &config.MaxMessageLength, &config.CreatedAt, &config.UpdatedAt)
	
	if err == nil {
		return &config, nil
	}
	
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query bridge config: %v", err)
	}

	// Create new config with defaults
	result, err := d.db.Exec(`
		INSERT INTO bridge_config (room_id, is_active, allow_media, allow_edits, allow_deletes, filter_words, max_message_length, created_at, updated_at) 
		VALUES (?, 1, 1, 1, 1, '[]', 4000, ?, ?)`,
		roomID, time.Now(), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create bridge config: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get bridge config ID: %v", err)
	}

	config = models.BridgeConfig{
		ID:               int(id),
		RoomID:           roomID,
		IsActive:         true,
		AllowMedia:       true,
		AllowEdits:       true,
		AllowDeletes:     true,
		FilterWords:      "[]",
		MaxMessageLength: 4000,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	return &config, nil
}

// GetAllActiveBridges returns all active bridge configurations with room mappings
func (d *Database) GetAllActiveBridges() (map[string][]*models.RoomMapping, error) {
	rows, err := d.db.Query(`
		SELECT rm.platform, rm.platform_room_id, rm.room_id, rm.room_name, rm.room_type,
			   rm.created_at, rm.updated_at
		FROM room_mappings rm
		INNER JOIN bridge_config bc ON rm.room_id = bc.room_id
		WHERE rm.is_active = 1 AND bc.is_active = 1
		ORDER BY rm.room_id, rm.platform`)
	if err != nil {
		return nil, fmt.Errorf("failed to query active bridges: %v", err)
	}
	defer rows.Close()

	bridges := make(map[string][]*models.RoomMapping)
	
	for rows.Next() {
		var mapping models.RoomMapping
		err := rows.Scan(&mapping.Platform, &mapping.PlatformRoomID, &mapping.RoomID,
			&mapping.RoomName, &mapping.RoomType, &mapping.CreatedAt, &mapping.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bridge mapping: %v", err)
		}

		mapping.IsActive = true
		bridges[mapping.PlatformRoomID] = append(bridges[mapping.PlatformRoomID], &mapping)
	}

	return bridges, nil
}
