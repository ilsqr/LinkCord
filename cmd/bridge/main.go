package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dcbot/internal/config"
	"dcbot/internal/database"
	"dcbot/internal/platforms/telegram"
	"dcbot/internal/platforms/discord"
	"dcbot/internal/bridge"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	fmt.Println("🌉 Cross-Platform Bridge Bot Starting...")
	fmt.Println("Telegram ↔ Discord")

	// Load configuration
	cfg := config.Load()
	
	// Initialize database
	fmt.Println("🗄️ Initializing database...")
	db, err := database.NewDatabase(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize bridge core
	fmt.Println("🌉 Initializing bridge core...")
	bridgeCore := bridge.NewBridgeCore(db)

	// Initialize platform clients based on configuration
	var telegramClient *telegram.Client
	var telegramHandler *telegram.MessageHandler
	var discordClient *discord.Client
	var discordHandler *discord.MessageHandler
	
	// Initialize Telegram if enabled
	if cfg.EnableTelegram {
		if cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
			log.Println("⚠️ Telegram is enabled but configuration is incomplete, skipping Telegram initialization")
		} else {
			fmt.Println("📱 Initializing Telegram bot...")
			telegramConfig := telegram.Config{
				BotToken: cfg.TelegramBotToken,
				ChatID:   cfg.TelegramChatID,
			}
			
			telegramClient, err = telegram.NewClient(telegramConfig)
			if err != nil {
				log.Printf("❌ Failed to create Telegram client: %v", err)
			} else {
				// Create message handler with bridge core and user mapping
				telegramHandler = telegram.NewMessageHandler(telegramClient, func(platform, chatID, userID, messageType, content string) error {
					// Set user mapping in bridge core for consistent usernames
					if username := telegramClient.GetUserDisplayName(userID); username != "" {
						bridgeCore.SetUserMapping(platform, userID, username)
					}
					return bridgeCore.ProcessMessageLegacy(platform, chatID, userID, messageType, content)
				})
				
				// Register Telegram platform with bridge core
				telegramAdapter := bridge.NewTelegramAdapter(telegramClient)
				bridgeCore.RegisterPlatform(telegramAdapter)
				
				// Start Telegram client
				if err := telegramClient.Start(telegramHandler.HandleMessage); err != nil {
					log.Printf("❌ Failed to start Telegram client: %v", err)
				}
			}
		}
	} else {
		fmt.Println("⏭️ Telegram is disabled in configuration")
	}

	// Initialize Discord if enabled
	if cfg.EnableDiscord {
		if cfg.DiscordBotToken == "" {
			log.Println("⚠️ Discord is enabled but bot token is missing, skipping Discord initialization")
		} else {
			fmt.Println("🎮 Initializing Discord bot...")
			discordClient, err = discord.NewClient(cfg.DiscordBotToken, cfg.DiscordGuildID)
			if err != nil {
				log.Printf("❌ Failed to create Discord client: %v", err)
			} else {
				// Create message handler with bridge core
				discordHandler = discord.NewMessageHandler(discordClient, func(platform, channelID, userID, messageType, content string) error {
					return bridgeCore.ProcessMessageLegacy(platform, channelID, userID, messageType, content)
				})
				
				// Register Discord platform with bridge core
				discordAdapter := bridge.NewDiscordAdapter(discordClient)
				bridgeCore.RegisterPlatform(discordAdapter)
				
				// Set bridge core reference in Discord handler
				discordHandler.SetBridgeCore(bridgeCore)
				
				// Set admin users (add your User ID here)
				discordHandler.SetAdminUsers([]string{
					"1359619658214412298", // Your Discord User ID
				})
				
				// Setup Discord handlers
				discordHandler.SetupHandlers()
				
				// Connect to Discord
				if err := discordClient.Connect(); err != nil {
					log.Printf("❌ Failed to connect to Discord: %v", err)
				}
			}
		}
	} else {
		fmt.Println("⏭️ Discord is disabled in configuration")
	}

	// Show active platforms
	showActivePlatforms(cfg)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	fmt.Println("✅ Bridge bot is running. Press Ctrl+C to stop.")
	<-stop

	fmt.Println("🛑 Shutting down bridge bot...")
	
	// Stop Telegram client if running
	if telegramClient != nil {
		telegramClient.Stop()
	}
	
	// Stop Discord client if running
	if discordClient != nil {
		discordClient.Disconnect()
	}
	
	fmt.Println("👋 Bridge bot stopped.")
}

// showActivePlatforms displays which platforms are active
func showActivePlatforms(cfg *config.Config) {
	fmt.Println("\n🔌 Active Platforms:")
	if cfg.EnableTelegram {
		fmt.Println("  ✅ Telegram")
	} else {
		fmt.Println("  ❌ Telegram (disabled)")
	}
	if cfg.EnableDiscord {
		fmt.Println("  ✅ Discord (Control Center)")
	} else {
		fmt.Println("  ❌ Discord (disabled)")
	}
	fmt.Println()
}
