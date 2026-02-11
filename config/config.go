package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration from environment variables
type Config struct {
	// Server
	ServerPort string
	BaseURL    string

	// OpenAI
	OpenAIAPIKey string
	OpenAIModel  string

	// Suno (via suno-api server)
	SunoBaseURL string

	// Telegram
	TelegramBotToken      string
	TelegramChatID        string
	TelegramWebhookPath   string
	TelegramWebhookSecret string
	TelegramWebhookURL    string

	// Workflow
	EnablePremiumFeatures bool
	MaxAudioSizeMB        int
}

// Load reads configuration from environment variables
func Load() *Config {
	return &Config{
		// Server
		ServerPort: getEnv("SERVER_PORT", "8080"),
		BaseURL:    getEnv("BASE_URL", "http://localhost:8080"),

		// OpenAI
		OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:  getEnv("OPENAI_MODEL", "gpt-4o"),

		// Suno (via suno-api server - see lib/suno/README.md for setup)
		SunoBaseURL: getEnv("SUNO_BASE_URL", "http://localhost:3000"),

		// Telegram
		TelegramBotToken:      getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:        getEnv("TELEGRAM_CHAT_ID", ""),
		TelegramWebhookPath:   getEnv("TELEGRAM_WEBHOOK_PATH", "/telegram/webhook"),
		TelegramWebhookSecret: getEnv("TELEGRAM_WEBHOOK_SECRET", ""),
		TelegramWebhookURL:    getEnv("TELEGRAM_WEBHOOK_URL", ""),

		// Workflow
		EnablePremiumFeatures: getEnvBool("ENABLE_PREMIUM_FEATURES", false),
		MaxAudioSizeMB:        getEnvInt("MAX_AUDIO_SIZE_MB", 50),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		i, err := strconv.Atoi(value)
		if err == nil {
			return i
		}
	}
	return defaultValue
}

