package deploy

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

const SYSTEMD_PATH = "/etc/systemd/system"
const TEMP_SERVICE_PATH = "/tmp"

// Setup performs remote VPS setup (called with --setup flag)
func Setup() error {
	slog.Info("Starting remote setup")

	// Load configuration from .env file in current directory
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	remotePath := cfg.RemotePath()

	// Step 0: Create installation directory with proper permissions
	if err := createDirectories(remotePath); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Step 1: Generate service file
	slog.Info("Generating systemd service file")
	serviceContent, err := GenerateServiceFile(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate service file: %w", err)
	}

	serviceName := getServiceName(cfg.AppName)

	// Write service file to temporary location
	tmpServicePath := fmt.Sprintf("%s/%s", TEMP_SERVICE_PATH, serviceName)
	if err := os.WriteFile(tmpServicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Step 2: Install/update systemd service (always update to ensure latest config)
	slog.Info("Installing/updating service file", "service", serviceName)
	if err := installService(tmpServicePath, serviceName); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	// Step 3: Check if service is enabled, and enable it if not
	serviceEnabled, err := checkServiceEnabled(serviceName)
	if err != nil {
		return fmt.Errorf("failed to check service status: %w", err)
	}

	if !serviceEnabled {
		slog.Info("Enabling service", "service", serviceName)
		if err := enableService(serviceName); err != nil {
			return fmt.Errorf("failed to enable service: %w", err)
		}
		slog.Info("Service enabled", "service", serviceName)
	} else {
		slog.Info("Service already enabled", "service", serviceName)
	}

	// Step 4: Restart or start the service
	if err := restartOrStartService(serviceName); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Step 5: Show status
	slog.Info("Service status")
	showServiceStatus(serviceName)

	// Cleanup temporary service file
	os.Remove(tmpServicePath)

	return nil
}

// checkServiceEnabled checks if a systemd service is enabled
func checkServiceEnabled(serviceName string) (bool, error) {
	// Check if service is enabled
	cmd := exec.Command("systemctl", "is-enabled", serviceName)
	output, _ := cmd.Output()

	enabled := strings.TrimSpace(string(output)) == "enabled"
	return enabled, nil
}

// createDirectories creates necessary directories with proper permissions
func createDirectories(remotePath string) error {
	uploadsPath := fmt.Sprintf("%s/uploads", remotePath)

	// Create uploads directory
	cmd := exec.Command("mkdir", "-p", uploadsPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create uploads directory: '%s' %w", uploadsPath, err)
	}

	return nil
}

// installService installs the systemd service file
func installService(tmpPath, serviceName string) error {
	servicePath := fmt.Sprintf("%s/%s", SYSTEMD_PATH, serviceName)

	cmd := exec.Command("sudo", "mv", tmpPath, servicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to move service file: '%s' %w", tmpPath, err)
	}

	// Reload systemd daemon
	cmd = exec.Command("sudo", "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: '%s' %w", serviceName, err)
	}

	slog.Info("Systemd service installed", "service", serviceName)
	return nil
}

// enableService enables the systemd service
func enableService(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "enable", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: '%s' %w", serviceName, err)
	}
	return nil
}

// restartOrStartService restarts or starts the service based on current state
func restartOrStartService(serviceName string) error {
	// Check if service is active
	cmd := exec.Command("systemctl", "is-active", "--quiet", serviceName)
	isActive := cmd.Run() == nil

	if isActive {
		slog.Info("Restarting service", "service", serviceName)
		cmd = exec.Command("sudo", "systemctl", "restart", serviceName)
	} else {
		slog.Info("Starting service", "service", serviceName)
		cmd = exec.Command("sudo", "systemctl", "start", serviceName)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start/restart service: '%s' %w", serviceName, err)
	}

	return nil
}

// showServiceStatus displays the service status
func showServiceStatus(serviceName string) {
	cmd := exec.Command("systemctl", "status", serviceName, "--no-pager", "-l")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // Ignore errors, just show status
}
