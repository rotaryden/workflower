package suno

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client handles Suno API communication
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Suno API client
func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 300 * time.Second, // Suno can take a while
		},
	}
}

// GenerateRequest represents a song generation request
type GenerateRequest struct {
	Lyrics             string  `json:"lyrics"`
	Style              string  `json:"style"`
	VocalType          string  `json:"vocal_type"`
	Weirdness          float64 `json:"weirdness"`
	StyleInfluence     string  `json:"style_influence,omitempty"`
	Persona            string  `json:"persona,omitempty"`
	Inspo              string  `json:"inspo,omitempty"`
	AudioReferencePath string  `json:"-"` // Not sent as JSON, uploaded separately
}

// GenerateResponse represents the Suno API response
type GenerateResponse struct {
	JobID   string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Generate submits a song generation request to Suno
func (c *Client) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// If there's an audio reference, use multipart upload
	if req.AudioReferencePath != "" {
		return c.generateWithAudio(ctx, req)
	}
	return c.generateJSON(ctx, req)
}

// generateJSON sends a JSON request without audio
func (c *Client) generateJSON(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// Prepare the request body according to Suno API format
	body := map[string]interface{}{
		"prompt":     req.Lyrics,
		"style":      req.Style,
		"vocal_type": req.VocalType,
		"weirdness":  req.Weirdness,
	}

	if req.StyleInfluence != "" {
		body["style_influence"] = req.StyleInfluence
	}
	if req.Persona != "" {
		body["persona"] = req.Persona
	}
	if req.Inspo != "" {
		body["inspo"] = req.Inspo
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate/v2/", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var genResp GenerateResponse
	if err := json.Unmarshal(respBody, &genResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &genResp, nil
}

// generateWithAudio sends a multipart request with audio file
func (c *Client) generateWithAudio(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// Open the audio file
	file, err := os.Open(req.AudioReferencePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the audio file
	part, err := writer.CreateFormFile("audio", filepath.Base(req.AudioReferencePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	// Add other fields
	fields := map[string]string{
		"prompt":     req.Lyrics,
		"style":      req.Style,
		"vocal_type": req.VocalType,
		"weirdness":  fmt.Sprintf("%.2f", req.Weirdness),
	}
	if req.StyleInfluence != "" {
		fields["style_influence"] = req.StyleInfluence
	}
	if req.Persona != "" {
		fields["persona"] = req.Persona
	}
	if req.Inspo != "" {
		fields["inspo"] = req.Inspo
	}

	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			return nil, fmt.Errorf("failed to write field %s: %w", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate/v2/", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var genResp GenerateResponse
	if err := json.Unmarshal(respBody, &genResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &genResp, nil
}

// GetStatus checks the status of a generation job
func (c *Client) GetStatus(ctx context.Context, jobID string) (*GenerateResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/feed/v2/?ids="+jobID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var genResp GenerateResponse
	if err := json.Unmarshal(respBody, &genResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &genResp, nil
}

