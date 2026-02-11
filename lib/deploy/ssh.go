package deploy

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHClient wraps an SSH connection
type SSHClient struct {
	client *ssh.Client
	config *Config
}

// NewSSHClient creates a new SSH client connection
func NewSSHClient(cfg *Config) (*SSHClient, error) {
	sshConfig, err := buildSSHConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build SSH config: %w", err)
	}

	// Parse host and port
	addr := cfg.RemoteHost
	if !strings.Contains(addr, ":") {
		addr = fmt.Sprintf("%s:%d", addr, cfg.SSHPort)
	}

	// Connect to remote server
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return &SSHClient{
		client: client,
		config: cfg,
	}, nil
}

// Close closes the SSH connection
func (c *SSHClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// RunCommand executes a command on the remote server
func (c *SSHClient) RunCommand(cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// RunCommandWithOutput executes a command and streams output to stdout/stderr
func (c *SSHClient) RunCommandWithOutput(cmd string) error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// CopyFile copies a local file to the remote server using SSH
func (c *SSHClient) CopyFile(localPath, remotePath string) error {
	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	// Create remote file via SSH command
	// First, write to temp file, then move to final location
	tmpPath := remotePath + ".tmp"
	
	// Write file content
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start cat command to write file
	if err := session.Start(fmt.Sprintf("cat > %s", tmpPath)); err != nil {
		return fmt.Errorf("failed to start cat command: %w", err)
	}

	// Write file data
	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write file data: %w", err)
	}
	stdin.Close()

	// Wait for cat to complete
	if err := session.Wait(); err != nil {
		return fmt.Errorf("failed to write remote file: %w", err)
	}

	// Move temp file to final location
	if _, err := c.RunCommand(fmt.Sprintf("mv %s %s", tmpPath, remotePath)); err != nil {
		return fmt.Errorf("failed to move file to final location: %w", err)
	}

	return nil
}

// MakeExecutable makes a remote file executable
func (c *SSHClient) MakeExecutable(remotePath string) error {
	_, err := c.RunCommand(fmt.Sprintf("chmod +x %s", remotePath))
	return err
}

// buildSSHConfig builds SSH client configuration
func buildSSHConfig(cfg *Config) (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	// Try custom key path first
	if cfg.SSHKeyPath != "" {
		key, err := loadPrivateKey(cfg.SSHKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load SSH key from %s: %w", cfg.SSHKeyPath, err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(key))
	}

	// Try SSH agent
	if agentAuth := getSSHAgentAuth(); agentAuth != nil {
		authMethods = append(authMethods, agentAuth)
	}

	// If no auth methods available, try default key locations
	if len(authMethods) == 0 {
		defaultKeys := []string{
			filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa"),
			filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519"),
		}
		for _, keyPath := range defaultKeys {
			if key, err := loadPrivateKey(keyPath); err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(key))
				break
			}
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no SSH authentication method available")
	}

	// Extract username from REMOTE_HOST (user@host format)
	user := "root" // default
	host := cfg.RemoteHost
	if strings.Contains(host, "@") {
		parts := strings.SplitN(host, "@", 2)
		user = parts[0]
		host = parts[1]
	}

	// Update config with parsed host
	cfg.RemoteHost = host

	return &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, should verify host keys
	}, nil
}

// loadPrivateKey loads a private key from file
func loadPrivateKey(path string) (ssh.Signer, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	return signer, nil
}

// getSSHAgentAuth returns SSH agent authentication method if available
func getSSHAgentAuth() ssh.AuthMethod {
	if sshAgent := os.Getenv("SSH_AUTH_SOCK"); sshAgent != "" {
		if conn, err := net.Dial("unix", sshAgent); err == nil {
			agentClient := agent.NewClient(conn)
			return ssh.PublicKeysCallback(agentClient.Signers)
		}
	}
	return nil
}
