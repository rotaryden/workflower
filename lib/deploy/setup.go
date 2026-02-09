package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Setup performs remote VPS setup (called with --setup flag)
func Setup() error {
	fmt.Println("🔧 Starting remote setup...")

	// Load configuration from .env file in current directory
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	remotePath := cfg.RemotePath()

	// Step 1: Generate service file
	fmt.Println("📝 Generating systemd service file...")
	serviceContent, err := GenerateServiceFile(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate service file: %w", err)
	}

	// Write service file to temporary location
	tmpServicePath := fmt.Sprintf("/tmp/%s.service", cfg.AppName)
	if err := os.WriteFile(tmpServicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Step 2: Check if service exists and is enabled
	serviceExists, err := checkServiceExists(cfg.AppName)
	if err != nil {
		return fmt.Errorf("failed to check service status: %w", err)
	}

	if !serviceExists {
		fmt.Printf("🔧 Setting up %s service...\n", cfg.AppName)

		// Create installation directory with proper permissions
		if err := createDirectories(remotePath, cfg); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}

		// Install systemd service
		if err := installService(tmpServicePath, cfg.AppName); err != nil {
			return fmt.Errorf("failed to install service: %w", err)
		}

		// Enable the service
		if err := enableService(cfg.AppName); err != nil {
			return fmt.Errorf("failed to enable service: %w", err)
		}

		fmt.Println("✅ Service enabled")
	} else {
		fmt.Println("✅ Service already configured and enabled")
		
		// Even if service exists, ensure directories have correct permissions
		if err := createDirectories(remotePath, cfg); err != nil {
			return fmt.Errorf("failed to ensure directories: %w", err)
		}
	}

	// Step 3: Restart or start the service
	if err := restartOrStartService(cfg.AppName); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Step 4: Show status
	fmt.Println()
	fmt.Println("📊 Service status:")
	showServiceStatus(cfg.AppName)

	// Cleanup temporary service file
	os.Remove(tmpServicePath)

	return nil
}

// checkServiceExists checks if a systemd service exists and is enabled
func checkServiceExists(serviceName string) (bool, error) {
	// Check if service unit file exists
	cmd := exec.Command("systemctl", "list-unit-files", fmt.Sprintf("%s.service", serviceName))
	output, _ := cmd.Output() // Don't fail if not found

	if !strings.Contains(string(output), serviceName) {
		return false, nil
	}

	// Check if service is enabled
	cmd = exec.Command("systemctl", "is-enabled", serviceName)
	output, _ = cmd.Output()

	enabled := strings.TrimSpace(string(output)) == "enabled"
	return enabled, nil
}

// createDirectories creates necessary directories with proper permissions
func createDirectories(remotePath string, cfg *Config) error {
	uploadsPath := fmt.Sprintf("%s/uploads", remotePath)

	// Create uploads directory
	cmd := exec.Command("sudo", "mkdir", "-p", uploadsPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create uploads directory: %w", err)
	}

	// Set ownership
	cmd = exec.Command("sudo", "chown", "-R",
		fmt.Sprintf("%s:%s", cfg.ServiceUser, cfg.ServiceGroup),
		remotePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set ownership: %w", err)
	}

	return nil
}

// installService installs the systemd service file
func installService(tmpPath, serviceName string) error {
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)

	cmd := exec.Command("sudo", "mv", tmpPath, servicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to move service file: %w", err)
	}

	// Reload systemd daemon
	cmd = exec.Command("sudo", "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	fmt.Println("✅ Systemd service installed")
	return nil
}

// enableService enables the systemd service
func enableService(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "enable", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	return nil
}

// restartOrStartService restarts or starts the service based on current state
func restartOrStartService(serviceName string) error {
	// Check if service is active
	cmd := exec.Command("systemctl", "is-active", "--quiet", serviceName)
	isActive := cmd.Run() == nil

	if isActive {
		fmt.Println("🔄 Restarting service...")
		cmd = exec.Command("sudo", "systemctl", "restart", serviceName)
	} else {
		fmt.Println("🚀 Starting service...")
		cmd = exec.Command("sudo", "systemctl", "start", serviceName)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start/restart service: %w", err)
	}

	return nil
}

// showServiceStatus displays the service status
func showServiceStatus(serviceName string) {
	cmd := exec.Command("sudo", "systemctl", "status", serviceName, "--no-pager", "-l")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // Ignore errors, just show status
}
