package deploy

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds deployment configuration
type Config struct {
	// Application settings
	AppName        string
	BaseRemotePath string

	// SSH settings
	RemoteHost string
	SSHPort    int
	SSHKeyPath string

	// Service settings
	ServiceUser        string
	ServiceGroup       string
	ServiceDescription string
}

// LoadConfig loads configuration from .env and .deploy.env files
func LoadConfig() (*Config, error) {
	// Load .env first
	if err := godotenv.Load(".env"); err != nil {
		return nil, fmt.Errorf("failed to load .env: %w", err)
	}

	// Load .deploy.env (overrides .env if conflicts)
	if err := godotenv.Load(".deploy.env"); err != nil {
		return nil, fmt.Errorf("failed to load .deploy.env: %w", err)
	}

	cfg := &Config{
		AppName:            os.Getenv("APP_NAME"),
		BaseRemotePath:     os.Getenv("BASE_REMOTE_PATH"),
		RemoteHost:         os.Getenv("REMOTE_HOST"),
		SSHPort:            22, // default
		SSHKeyPath:         os.Getenv("SSH_KEY_PATH"),
		ServiceUser:        getEnvOrDefault("SERVICE_USER", "www-data"),
		ServiceGroup:       getEnvOrDefault("SERVICE_GROUP", "www-data"),
		ServiceDescription: getEnvOrDefault("SERVICE_DESCRIPTION", "Suno Workflow Server"),
	}

	// Parse SSH port if provided
	if portStr := os.Getenv("SSH_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SSH_PORT: %w", err)
		}
		cfg.SSHPort = port
	}

	// Validate required fields
	if cfg.RemoteHost == "" {
		return nil, fmt.Errorf("REMOTE_HOST not set in .deploy.env")
	}
	if cfg.BaseRemotePath == "" {
		return nil, fmt.Errorf("BASE_REMOTE_PATH not set in .env")
	}
	if cfg.AppName == "" {
		return nil, fmt.Errorf("APP_NAME not set in .env")
	}

	return cfg, nil
}

// RemotePath returns the full remote path for the application
func (c *Config) RemotePath() string {
	return fmt.Sprintf("%s/%s", c.BaseRemotePath, c.AppName)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
