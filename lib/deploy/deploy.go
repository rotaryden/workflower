package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Deploy performs the full deployment workflow
func Deploy() error {
	fmt.Println("📝 Loading environment variables...")
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Step 1: Build binary for Linux
	fmt.Println("🚀 Building for Linux...")
	if err := buildLinuxBinary(cfg.AppName); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	defer cleanupBinary(cfg.AppName)

	// Step 2: Establish SSH connection
	fmt.Printf("📦 Deploying to %s:%s...\n", cfg.RemoteHost, cfg.RemotePath())
	client, err := NewSSHClient(cfg)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer client.Close()

	remotePath := cfg.RemotePath()

	// Step 3: Ensure remote directory exists
	// Try without sudo first (if directory exists with correct permissions)
	mkdirCmd := fmt.Sprintf("mkdir -p %s", remotePath)
	_, err = client.RunCommand(mkdirCmd)
	if err != nil {
		// If that fails, try with sudo
		mkdirCmd = fmt.Sprintf("sudo mkdir -p %s && sudo chown %s:%s %s",
			remotePath, cfg.ServiceUser, cfg.ServiceGroup, remotePath)
		output, err := client.RunCommand(mkdirCmd)
		if err != nil {
			return fmt.Errorf("failed to create remote directory (ensure user has sudo NOPASSWD or create directory manually): %s: %w", output, err)
		}
	}

	// Step 4: Copy binary
	fmt.Println("📤 Copying binary...")
	binaryPath := filepath.Join(remotePath, cfg.AppName)
	if err := client.CopyFile(cfg.AppName, binaryPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Step 5: Copy .env file if it exists
	if fileExists(".env") {
		fmt.Println("📝 Copying .env file...")
		envPath := filepath.Join(remotePath, ".env")
		if err := client.CopyFile(".env", envPath); err != nil {
			return fmt.Errorf("failed to copy .env: %w", err)
		}
	}

	// Step 6: Copy .deploy.env if it exists
	if fileExists(".deploy.env") {
		fmt.Println("📝 Copying .deploy.env file...")
		envExamplePath := filepath.Join(remotePath, ".deploy.env")
		if err := client.CopyFile(".deploy.env", envExamplePath); err != nil {
			return fmt.Errorf("failed to copy .deploy.env: %w", err)
		}
	}

	// Step 7: Make binary executable
	fmt.Println("🔧 Making binary executable...")
	if err := client.MakeExecutable(binaryPath); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Step 8: Run remote setup
	fmt.Println("🔧 Running remote setup...")
	setupCmd := fmt.Sprintf("cd %s && ./%s --setup", remotePath, cfg.AppName)
	if err := client.RunCommandWithOutput(setupCmd); err != nil {
		return fmt.Errorf("remote setup failed: %w", err)
	}

	// Success!
	fmt.Println()
	fmt.Println("✅ Deployment complete!")
	fmt.Println()
	fmt.Println("📋 Useful commands:")
	fmt.Printf("  View logs: ssh %s 'sudo journalctl -u %s -f'\n", cfg.RemoteHost, cfg.AppName)
	fmt.Printf("  Check status: ssh %s 'sudo systemctl status %s'\n", cfg.RemoteHost, cfg.AppName)
	fmt.Printf("  Edit .env: ssh %s 'sudo nano %s/.env && sudo systemctl restart %s'\n",
		cfg.RemoteHost, remotePath, cfg.AppName)

	return nil
}

// buildLinuxBinary builds the application for Linux AMD64
func buildLinuxBinary(appName string) error {
	cmd := exec.Command("go", "build", "-o", appName, ".")
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH=amd64",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	return nil
}

// cleanupBinary removes the locally built binary
func cleanupBinary(appName string) {
	if err := os.Remove(appName); err != nil && !os.IsNotExist(err) {
		fmt.Printf("⚠️  Failed to cleanup binary: %v\n", err)
	}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
