package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles OpenAI API communication
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new OpenAI client
func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.openai.com/v1",
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the OpenAI chat completion request
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// ChatResponse represents the OpenAI chat completion response
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Chat sends a chat completion request and returns the response
func (c *Client) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	return c.ChatWithMessages(ctx, messages)
}

// ChatWithMessages sends a chat completion request with custom messages
func (c *Client) ChatWithMessages(ctx context.Context, messages []Message) (string, error) {
	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// Prompt templates for the workflow

const LyricsGenerationPrompt = `You are a talented songwriter and lyricist. Your task is to create compelling, emotionally resonant song lyrics based on the given description.

Guidelines:
- Create lyrics that capture the essence and emotion of the subject
- Structure the lyrics with verses, chorus, and optionally a bridge
- Use vivid imagery and poetic language
- Make the lyrics singable with good rhythm and flow
- Keep the total length appropriate for a 3-4 minute song

Output ONLY the lyrics text, no explanations or metadata.`

const SunoPropertiesPrompt = `You are an expert music producer helping to configure AI music generation. 
Based on the subject description and lyrics, determine the optimal Suno properties.

Output a JSON object with these fields:
{
  "style": "genre/style description (e.g., 'upbeat pop with electronic elements', 'melancholic indie folk')",
  "vocal_type": "vocal configuration (e.g., 'female soprano', 'male baritone', 'duet male and female')",
  "lyrics_mode": "default or custom",
  "weirdness": number from 0.0 to 1.0 (how experimental the sound should be),
  "style_influence": "specific artist or style influences if applicable"
}

Output ONLY the JSON object, no explanations.`

const BracketInstructionsPrompt = `You are an expert in Suno AI music generation. 
Your task is to add bracket instructions to the lyrics for better vocal and musical interpretation.

Available bracket instructions:
- [Verse], [Chorus], [Bridge], [Outro], [Intro]
- [Male Voice], [Female Voice], [Duet]
- [Spoken], [Whispered], [Belted], [Falsetto]
- [Instrumental Break], [Guitar Solo], [Piano Intro]
- [Slow], [Fast], [Building], [Dropping]
- [Emotional], [Energetic], [Calm], [Intense]

Take the original lyrics and insert appropriate bracket instructions throughout. 
Maintain the original lyrics but enhance them with these production cues.

Output ONLY the enhanced lyrics with bracket instructions, no explanations.`

const PersonaInspoPrompt = `You are an expert in Suno AI music generation. Based on the song style and subject, suggest:

1. A "Persona" - a fictional artist persona that matches the style (describe their voice, style, background)
2. An "Inspo" - specific artists or songs that should inspire the generation

Output a JSON object:
{
  "persona": "description of the fictional artist persona",
  "inspo": "comma-separated list of artist names or song references"
}

Output ONLY the JSON object, no explanations.`
