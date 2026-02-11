package suno

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles Suno API communication via the third-party suno-api server
// This wraps the unofficial suno-api (https://github.com/gcui-art/suno-api)
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Suno API client
// baseURL should point to your suno-api server (e.g., "http://localhost:3000")
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 300 * time.Second, // Suno generation can take a while
		},
	}
}

// GenerateRequest represents a simple song generation request using a prompt
type GenerateRequest struct {
	Prompt           string `json:"prompt"`
	MakeInstrumental bool   `json:"make_instrumental"`
	Model            string `json:"model,omitempty"` // Default: "chirp-v3-5", also supports "chirp-v3-0"
	WaitAudio        bool   `json:"wait_audio"`
}

// CustomGenerateRequest represents a custom song generation request with full control
type CustomGenerateRequest struct {
	Prompt           string `json:"prompt"`            // Lyrics or detailed prompt
	Tags             string `json:"tags"`              // Music style/genre
	NegativeTags     string `json:"negative_tags,omitempty"` // Negative music genre
	Title            string `json:"title"`
	MakeInstrumental bool   `json:"make_instrumental,omitempty"`
	Model            string `json:"model,omitempty"` // Default: "chirp-v3-5"
	WaitAudio        bool   `json:"wait_audio,omitempty"`
}

// ExtendAudioRequest represents a request to extend audio length
type ExtendAudioRequest struct {
	AudioID    string `json:"audio_id"`
	Prompt     string `json:"prompt,omitempty"`      // Additional lyrics
	ContinueAt string `json:"continue_at,omitempty"` // Extend from mm:ss (e.g., "00:30")
	Title      string `json:"title,omitempty"`
	Tags       string `json:"tags,omitempty"`
	NegativeTags string `json:"negative_tags,omitempty"`
	Model      string `json:"model,omitempty"`
}

// GenerateStemsRequest represents a request to generate stem tracks
type GenerateStemsRequest struct {
	AudioID string `json:"audio_id"`
}

// GenerateLyricsRequest represents a request to generate lyrics
type GenerateLyricsRequest struct {
	Prompt string `json:"prompt"`
}

// ConcatRequest represents a request to concatenate audio clips
type ConcatRequest struct {
	ClipID string `json:"clip_id"`
}

// AudioInfo represents the detailed audio information from Suno API
// Matches the audio_info schema from the Swagger spec
type AudioInfo struct {
	ID                   string  `json:"id"`
	Title                string  `json:"title"`
	ImageURL             string  `json:"image_url"`
	Lyric                string  `json:"lyric"`
	AudioURL             string  `json:"audio_url"`
	VideoURL             string  `json:"video_url"`
	CreatedAt            string  `json:"created_at"`
	ModelName            string  `json:"model_name"`
	Status               string  `json:"status"` // "submitted", "queue", "streaming", "complete"
	GPTDescriptionPrompt string  `json:"gpt_description_prompt"`
	Prompt               string  `json:"prompt"`
	Type                 string  `json:"type"`
	Tags                 string  `json:"tags"`
	Duration             float64 `json:"duration,omitempty"`
}

// GenerateResponse is an alias for AudioInfo for backward compatibility
type GenerateResponse = AudioInfo

// LyricsResponse represents the response from generate_lyrics endpoint
type LyricsResponse struct {
	Text   string `json:"text"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// PersonaClip represents a clip in a persona
type PersonaClip struct {
	Clip any `json:"clip"`
}

// Persona represents persona information
type Persona struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	ImageS3ID      string        `json:"image_s3_id"`
	RootClipID     string        `json:"root_clip_id"`
	Clip           any           `json:"clip"`
	PersonaClips   []PersonaClip `json:"persona_clips"`
	IsSunoPersona  bool          `json:"is_suno_persona"`
	IsPublic       bool          `json:"is_public"`
	UpvoteCount    int           `json:"upvote_count"`
	ClipCount      int           `json:"clip_count"`
}

// PersonaResponse represents the response from get persona endpoint
type PersonaResponse struct {
	Persona      Persona `json:"persona"`
	TotalResults int     `json:"total_results"`
	CurrentPage  int     `json:"current_page"`
	IsFollowing  bool    `json:"is_following"`
}

// QuotaInfo represents the account quota information
type QuotaInfo struct {
	CreditsLeft   int    `json:"credits_left"`
	Period        string `json:"period"`
	MonthlyLimit  int    `json:"monthly_limit"`
	MonthlyUsage  int    `json:"monthly_usage"`
}

// Generate submits a simple song generation request using a text prompt
// It will automatically fill in the lyrics. 2 audio files will be generated, consuming 10 credits total.
// Returns a slice of AudioInfo (typically 2 variations)
func (c *Client) Generate(ctx context.Context, req *GenerateRequest) ([]AudioInfo, error) {
	return c.doPost(ctx, "/api/generate", req)
}

// CustomGenerate submits a custom song generation request with full control over lyrics, style, and title
// 2 audio files will be generated for each request, consuming 10 credits total.
// Returns a slice of AudioInfo (typically 2 variations)
func (c *Client) CustomGenerate(ctx context.Context, req *CustomGenerateRequest) ([]AudioInfo, error) {
	return c.doPost(ctx, "/api/custom_generate", req)
}

// ExtendAudio extends the length of an existing audio clip
func (c *Client) ExtendAudio(ctx context.Context, req *ExtendAudioRequest) ([]AudioInfo, error) {
	return c.doPost(ctx, "/api/extend_audio", req)
}

// GenerateStems generates stem tracks (separate audio and music tracks)
func (c *Client) GenerateStems(ctx context.Context, req *GenerateStemsRequest) (*AudioInfo, error) {
	var result AudioInfo
	err := c.doPostSingle(ctx, "/api/generate_stems", req, &result)
	return &result, err
}

// GenerateLyrics generates lyrics based on a prompt
func (c *Client) GenerateLyrics(ctx context.Context, req *GenerateLyricsRequest) (*LyricsResponse, error) {
	var result LyricsResponse
	err := c.doPostSingle(ctx, "/api/generate_lyrics", req, &result)
	return &result, err
}

// Concat generates the whole song from extensions
func (c *Client) Concat(ctx context.Context, req *ConcatRequest) (*AudioInfo, error) {
	var result AudioInfo
	err := c.doPostSingle(ctx, "/api/concat", req, &result)
	return &result, err
}

// Get retrieves audio information by ID(s)
// Pass comma-separated IDs to get multiple tracks, or empty string to get all
// Optionally specify page number for pagination (default: 0 means no pagination)
func (c *Client) Get(ctx context.Context, ids string, page int) ([]AudioInfo, error) {
	url := c.baseURL + "/api/get"
	
	if ids != "" {
		url += "?ids=" + ids
	}
	
	if page > 0 {
		if ids != "" {
			url += fmt.Sprintf("&page=%d", page)
		} else {
			url += fmt.Sprintf("?page=%d", page)
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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

	var result []AudioInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// GetClip retrieves clip information by ID
func (c *Client) GetClip(ctx context.Context, id string) (*AudioInfo, error) {
	url := fmt.Sprintf("%s/api/clip?id=%s", c.baseURL, id)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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

	var result AudioInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetAlignedLyrics retrieves lyric alignment for a song
func (c *Client) GetAlignedLyrics(ctx context.Context, songID string) (*AudioInfo, error) {
	url := fmt.Sprintf("%s/api/get_aligned_lyrics?song_id=%s", c.baseURL, songID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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

	var result AudioInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetPersona retrieves persona information including associated clips
func (c *Client) GetPersona(ctx context.Context, id string, page int) (*PersonaResponse, error) {
	url := fmt.Sprintf("%s/api/persona?id=%s", c.baseURL, id)
	if page > 0 {
		url += fmt.Sprintf("&page=%d", page)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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

	var result PersonaResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetQuota retrieves the current account quota information
func (c *Client) GetQuota(ctx context.Context) (*QuotaInfo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/get_limit", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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

	var quotaInfo QuotaInfo
	if err := json.Unmarshal(respBody, &quotaInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &quotaInfo, nil
}

// WaitForCompletion polls the API until the audio with the given ID is ready
// It checks every pollInterval until the status is "streaming" or "complete"
// Returns an error if the context is cancelled or if max retries are exceeded
func (c *Client) WaitForCompletion(ctx context.Context, id string, pollInterval time.Duration, maxRetries int) (*AudioInfo, error) {
	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		responses, err := c.Get(ctx, id, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to get audio info: %w", err)
		}

		if len(responses) == 0 {
			return nil, fmt.Errorf("no audio found with ID: %s", id)
		}

		audio := &responses[0]
		if audio.Status == "streaming" || audio.Status == "complete" {
			return audio, nil
		}

		time.Sleep(pollInterval)
	}

	return nil, fmt.Errorf("max retries exceeded waiting for audio completion")
}

// doPost is a helper method for POST requests that return an array of AudioInfo
func (c *Client) doPost(ctx context.Context, endpoint string, reqBody any) ([]AudioInfo, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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

	var result []AudioInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// doPostSingle is a helper method for POST requests that return a single object
func (c *Client) doPostSingle(ctx context.Context, endpoint string, reqBody any, result any) error {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

