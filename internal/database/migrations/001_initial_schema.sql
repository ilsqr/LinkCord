-- Migration: 001_initial_schema.sql
-- Description: Initial database schema for DCBot bridge
-- Created: 2025-06-29

-- Users table: Core user entities
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User mappings: Link users across platforms
CREATE TABLE IF NOT EXISTS user_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    platform TEXT NOT NULL CHECK(platform IN ('telegram', 'discord')),
    platform_user_id TEXT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(platform, platform_user_id)
);

-- Rooms table: Core room/channel entities  
CREATE TABLE IF NOT EXISTS rooms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Room mappings: Link rooms/channels across platforms
CREATE TABLE IF NOT EXISTS room_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id INTEGER NOT NULL,
    platform TEXT NOT NULL CHECK(platform IN ('telegram', 'discord')),
    platform_room_id TEXT NOT NULL,
    room_name TEXT NOT NULL DEFAULT '',
    room_type TEXT NOT NULL DEFAULT 'channel' CHECK(room_type IN ('channel', 'group', 'dm')),
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    UNIQUE(platform, platform_room_id)
);

-- Messages: Store bridged messages
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    original_id TEXT NOT NULL,
    source_platform TEXT NOT NULL CHECK(source_platform IN ('telegram', 'discord')),
    source_room_id TEXT NOT NULL,
    source_user_id TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    message_type TEXT NOT NULL DEFAULT 'text' CHECK(message_type IN ('text', 'image', 'file', 'audio', 'video', 'sticker')),
    media_url TEXT NOT NULL DEFAULT '',
    media_mime_type TEXT NOT NULL DEFAULT '',
    reply_to_id INTEGER,
    is_edited BOOLEAN NOT NULL DEFAULT 0,
    is_deleted BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (reply_to_id) REFERENCES messages(id),
    UNIQUE(source_platform, original_id)
);

-- Message mappings: Track how messages are mapped across platforms
CREATE TABLE IF NOT EXISTS message_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id INTEGER NOT NULL,
    platform TEXT NOT NULL CHECK(platform IN ('telegram', 'discord')),
    platform_msg_id TEXT NOT NULL,
    platform_room_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'sent', 'failed', 'deleted')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    UNIQUE(platform, platform_msg_id)
);

-- Bridge configuration: Per-room bridge settings
CREATE TABLE IF NOT EXISTS bridge_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    allow_media BOOLEAN NOT NULL DEFAULT 1,
    allow_edits BOOLEAN NOT NULL DEFAULT 1,
    allow_deletes BOOLEAN NOT NULL DEFAULT 1,
    filter_words TEXT NOT NULL DEFAULT '[]', -- JSON array
    max_message_length INTEGER NOT NULL DEFAULT 4000,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    UNIQUE(room_id)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_user_mappings_platform_user_id ON user_mappings(platform, platform_user_id);
CREATE INDEX IF NOT EXISTS idx_room_mappings_platform_room_id ON room_mappings(platform, platform_room_id);
CREATE INDEX IF NOT EXISTS idx_messages_source ON messages(source_platform, source_room_id);
CREATE INDEX IF NOT EXISTS idx_message_mappings_platform ON message_mappings(platform, platform_msg_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_reply_to ON messages(reply_to_id) WHERE reply_to_id IS NOT NULL;
