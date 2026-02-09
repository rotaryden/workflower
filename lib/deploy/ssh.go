package deploy

import (
	"fmt"
	"io"
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

// CopyFile copies a local file to the remote server using SCP
func (c *SSHClient) CopyFile(localPath, remotePath string) error {
	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Create SCP session
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Set up pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start SCP in sink mode
	if err := session.Start(fmt.Sprintf("scp -t %s", remotePath)); err != nil {
		return fmt.Errorf("failed to start scp: %w", err)
	}

	// Send file header
	fmt.Fprintf(stdin, "C%04o %d %s\n", fileInfo.Mode().Perm(), fileInfo.Size(), filepath.Base(remotePath))

	// Copy file content
	if _, err := io.Copy(stdin, file); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Send null byte to indicate end
	fmt.Fprint(stdin, "\x00")

	stdin.Close()

	// Wait for session to complete
	if err := session.Wait(); err != nil {
		return fmt.Errorf("scp failed: %w", err)
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
