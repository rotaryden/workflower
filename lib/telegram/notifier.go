package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Notifier handles Telegram notifications
type Notifier struct {
	botToken   string
	chatID     string
	httpClient *http.Client
}

// NewNotifier creates a new Telegram notifier
func NewNotifier(botToken, chatID string) *Notifier {
	return &Notifier{
		botToken: botToken,
		chatID:   chatID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendMessageRequest represents a Telegram sendMessage request
type SendMessageRequest struct {
	ChatID      string      `json:"chat_id"`
	Text        string      `json:"text"`
	ParseMode   string      `json:"parse_mode,omitempty"`
	ReplyMarkup interface{} `json:"reply_markup,omitempty"`
}

// TelegramResponse represents the Telegram API response
type TelegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	Result      struct {
		MessageID int `json:"message_id"`
	} `json:"result,omitempty"`
}

// Send sends a message to the configured Telegram chat
func (n *Notifier) Send(ctx context.Context, message string) error {
	return n.sendMessage(ctx, SendMessageRequest{
		ChatID:    n.chatID,
		Text:      message,
		ParseMode: "HTML",
	})
}

// SendToChat sends a message to a specific Telegram chat
func (n *Notifier) SendToChat(ctx context.Context, chatID, message string) error {
	return n.sendMessage(ctx, SendMessageRequest{
		ChatID:    chatID,
		Text:      message,
		ParseMode: "HTML",
	})
}

// SendWithLink sends a message with an inline keyboard button link
func (n *Notifier) SendWithLink(ctx context.Context, message, buttonText, buttonURL string) error {
	// Create inline keyboard with link button
	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{
					"text": buttonText,
					"url":  buttonURL,
				},
			},
		},
	}

	return n.sendMessage(ctx, SendMessageRequest{
		ChatID:      n.chatID,
		Text:        message,
		ParseMode:   "HTML",
		ReplyMarkup: keyboard,
	})
}

type setWebhookRequest struct {
	URL            string   `json:"url"`
	SecretToken    string   `json:"secret_token,omitempty"`
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}

type telegramBoolResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	Result      bool   `json:"result,omitempty"`
}

// SetWebhook registers a Telegram webhook URL
func (n *Notifier) SetWebhook(ctx context.Context, webhookURL, secretToken string) error {
	if n.botToken == "" {
		return nil
	}
	if webhookURL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	reqBody := setWebhookRequest{
		URL:            webhookURL,
		SecretToken:    secretToken,
		AllowedUpdates: []string{"message", "edited_message"},
	}

	body, err := n.doRequest(ctx, "setWebhook", reqBody)
	if err != nil {
		return err
	}

	var tgResp telegramBoolResponse
	if err := json.Unmarshal(body, &tgResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !tgResp.OK {
		return fmt.Errorf("telegram API error: %s", tgResp.Description)
	}

	return nil
}

func (n *Notifier) sendMessage(ctx context.Context, reqBody SendMessageRequest) error {
	if n.botToken == "" || reqBody.ChatID == "" {
		// Silent skip if not configured
		return nil
	}

	body, err := n.doRequest(ctx, "sendMessage", reqBody)
	if err != nil {
		return err
	}

	var tgResp TelegramResponse
	if err := json.Unmarshal(body, &tgResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !tgResp.OK {
		return fmt.Errorf("telegram API error: %s", tgResp.Description)
	}

	return nil
}

func (n *Notifier) doRequest(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", n.botToken, endpoint)

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

