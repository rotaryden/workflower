package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"workflower/config"
	"workflower/handlers"
	"workflower/storage"
	"workflower/telegram"
	"workflower/templates"
	"workflower/workflow"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	useTunnel := flag.Bool("L", false, "Start Cloudflare tunnel and override BASE_URL/TELEGRAM_WEBHOOK_URL")
	flag.Parse()

	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	if *useTunnel {
		tunnelURL, err := startCloudflareTunnel(context.Background(), cfg.ServerPort)
		if err != nil {
			log.Fatalf("Failed to start Cloudflare tunnel: %v", err)
		}

		baseURL := strings.TrimRight(tunnelURL, "/")
		cfg.BaseURL = baseURL

		webhookPath := strings.TrimSpace(cfg.TelegramWebhookPath)
		if webhookPath == "" {
			webhookPath = "/telegram/webhook"
		} else if !strings.HasPrefix(webhookPath, "/") {
			webhookPath = "/" + webhookPath
		}
		cfg.TelegramWebhookURL = cfg.BaseURL + webhookPath

		log.Printf("🌐 Cloudflare tunnel active: %s", cfg.BaseURL)
		log.Printf("🔔 Telegram webhook URL: %s", cfg.TelegramWebhookURL)
	}

	// Validate required configuration
	if cfg.OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	// Initialize templates
	if err := templates.Init(); err != nil {
		log.Fatalf("Failed to initialize templates: %v", err)
	}

	// Initialize storage
	store := storage.NewStore()

	// Initialize workflow engine
	engine := workflow.NewEngine(cfg, store)

	// Initialize handlers
	handler := handlers.NewHandler(cfg, store, engine)

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(handlers.ErrorHandler())

	// Register routes
	handler.RegisterRoutes(r)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("🎵 Suno Workflow Server starting on http://localhost%s", addr)
	log.Printf("📝 OpenAI Model: %s", cfg.OpenAIModel)
	if cfg.TelegramBotToken != "" {
		log.Printf("📱 Telegram notifications enabled")
		log.Printf("🔔 Telegram webhook path: %s", cfg.TelegramWebhookPath)
		if cfg.TelegramWebhookURL != "" {
			notifier := telegram.NewNotifier(cfg.TelegramBotToken, cfg.TelegramChatID)
			if err := notifier.SetWebhook(context.Background(), cfg.TelegramWebhookURL, cfg.TelegramWebhookSecret); err != nil {
				log.Printf("⚠️  Failed to set Telegram webhook: %v", err)
			} else {
				log.Printf("✅ Telegram webhook registered: %s", cfg.TelegramWebhookURL)
			}
		}
	}
	if cfg.EnablePremiumFeatures {
		log.Printf("⭐ Premium features enabled by default")
	}

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

