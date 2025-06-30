package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Platform Enable/Disable flags
	EnableTelegram bool
	EnableDiscord  bool

	// Telegram configuration
	TelegramBotToken string
	TelegramChatID   string

	// Discord configuration
	DiscordBotToken string
	DiscordGuildID  string
	DiscordChannelID string

	// Database configuration
	DatabasePath string

	// Logging configuration
	LogLevel string
	LogFile  string

	// API configuration
	APIPort   int
	APIEnable bool
}

func Load() *Config {
	apiPort, _ := strconv.Atoi(getEnv("API_PORT", "8080"))
	apiEnable, _ := strconv.ParseBool(getEnv("API_ENABLE", "false"))
	
	// Platform enable/disable flags
	enableTelegram, _ := strconv.ParseBool(getEnv("ENABLE_TELEGRAM", "true"))
	enableDiscord, _ := strconv.ParseBool(getEnv("ENABLE_DISCORD", "true"))

	return &Config{
		EnableTelegram: enableTelegram,
		EnableDiscord:  enableDiscord,

		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),

		DiscordBotToken:  getEnv("DISCORD_BOT_TOKEN", ""),
		DiscordGuildID:   getEnv("DISCORD_GUILD_ID", ""),
		DiscordChannelID: getEnv("DISCORD_CHANNEL_ID", ""),

		DatabasePath: getEnv("DATABASE_PATH", "./bridge.db"),

		LogLevel: getEnv("LOG_LEVEL", "info"),
		LogFile:  getEnv("LOG_FILE", "./logs/bridge.log"),

		APIPort:   apiPort,
		APIEnable: apiEnable,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
