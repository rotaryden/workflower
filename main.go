package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"workflower/config"
	"workflower/lib/deploy"
	"workflower/handlers"
	"workflower/lib/logger"
	"workflower/lib/telegram"
	"workflower/storage"
	"workflower/templates/prompts"
	"workflower/templates/ui_templates"
	"workflower/workflow"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Initialize logger
	logger.Init()

	deployFlag := flag.Bool("D", false, "Deploy to remote server")
	setupFlag := flag.Bool("setup", false, "Run remote setup (used during deployment)")
	useTunnel := flag.Bool("L", false, "Start Cloudflare tunnel and override BASE_URL/TELEGRAM_WEBHOOK_URL")
	flag.Parse()

	// Handle deployment mode
	if *deployFlag {
		if err := deploy.Deploy(); err != nil {
			slog.Error("Deployment failed", "error", err)
			os.Exit(1)
		}
		return
	}

	// Handle remote setup mode
	if *setupFlag {
		if err := deploy.Setup(); err != nil {
			slog.Error("Setup failed", "error", err)
			os.Exit(1)
		}
		return
	}

	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	if *useTunnel {
		tunnelURL, err := deploy.StartCloudflareTunnel(context.Background(), cfg.ServerPort)
		if err != nil {
			slog.Error("Failed to start Cloudflare tunnel", "error", err)
			os.Exit(1)
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

		slog.Info("Cloudflare tunnel active", "url", cfg.BaseURL)
		slog.Info("Telegram webhook URL configured", "url", cfg.TelegramWebhookURL)
	}

	// Validate required configuration
	if cfg.OpenAIAPIKey == "" {
		slog.Error("OPENAI_API_KEY is required")
		os.Exit(1)
	}

	// Initialize templates
	templates, err := ui_templates.Init()
	if err != nil {
		slog.Error("Failed to initialize templates", "error", err)
		os.Exit(1)
	}

	// Initialize prompts
	promptsList := prompts.Init()

	// Initialize storage
	store := storage.NewStore()

	// Initialize workflow engine
	engine := workflow.NewEngine(cfg, store, promptsList)

	// Initialize handlers
	handler := handlers.NewHandler(cfg, store, engine, templates)

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
	slog.Info("Suno Workflow Server starting", "address", fmt.Sprintf("http://localhost%s", addr))
	slog.Info("OpenAI configuration", "model", cfg.OpenAIModel)
	if cfg.TelegramBotToken != "" {
		slog.Info("Telegram notifications enabled")
		slog.Info("Telegram webhook path configured", "path", cfg.TelegramWebhookPath)
		if cfg.TelegramWebhookURL != "" {
			notifier := telegram.NewNotifier(cfg.TelegramBotToken, cfg.TelegramChatID)
			if err := notifier.SetWebhook(context.Background(), cfg.TelegramWebhookURL, cfg.TelegramWebhookSecret); err != nil {
				slog.Warn("Failed to set Telegram webhook", "error", err)
			} else {
				slog.Info("Telegram webhook registered", "url", cfg.TelegramWebhookURL)
			}
		}
	}
	if cfg.EnablePremiumFeatures {
		slog.Info("Premium features enabled by default")
	}

	if err := r.Run(addr); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
