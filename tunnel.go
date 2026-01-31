package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var tunnelURLRegex = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

func startCloudflareTunnel(ctx context.Context, port string) (string, error) {
	if port == "" {
		port = "8080"
	}

	if _, err := exec.LookPath("cloudflared"); err != nil {
		return "", fmt.Errorf("cloudflared not found in PATH: %w", err)
	}

	cmd := exec.CommandContext(ctx, "cloudflared", "tunnel", "--url", fmt.Sprintf("http://localhost:%s", port))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get cloudflared stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get cloudflared stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start cloudflared: %w", err)
	}

	urlCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go scanTunnelOutput(stdout, urlCh)
	go scanTunnelOutput(stderr, urlCh)

	go func() {
		errCh <- cmd.Wait()
	}()

	timeout := time.NewTimer(25 * time.Second)
	defer timeout.Stop()

	select {
	case url := <-urlCh:
		return strings.TrimRight(url, "/"), nil
	case err := <-errCh:
		if err == nil {
			err = fmt.Errorf("cloudflared exited without error")
		}
		return "", err
	case <-timeout.C:
		return "", fmt.Errorf("timed out waiting for cloudflared URL")
	}
}

func scanTunnelOutput(reader io.Reader, urlCh chan<- string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if url := extractTunnelURL(line); url != "" {
			select {
			case urlCh <- url:
			default:
			}
		}
	}
}

func extractTunnelURL(line string) string {
	if line == "" {
		return ""
	}
	return tunnelURLRegex.FindString(line)
}
