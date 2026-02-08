package telegram

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// WebhookSecretHeader is the header Telegram sends when secret token is set.
const WebhookSecretHeader = "X-Telegram-Bot-Api-Secret-Token"

// Update represents a Telegram webhook update.
type Update struct {
	UpdateID      int            `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	EditedMessage *Message       `json:"edited_message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID int    `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      Chat   `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text,omitempty"`
	Caption   string `json:"caption,omitempty"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Title     string `json:"title,omitempty"`
}

// User represents a Telegram user.
type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

// CallbackQuery represents a Telegram callback query.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from,omitempty"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

// VerifyWebhookSecret validates the Telegram webhook secret token.
func VerifyWebhookSecret(r *http.Request, expectedSecret string) bool {
	expected := strings.TrimSpace(expectedSecret)
	if expected == "" {
		return true
	}

	provided := r.Header.Get(WebhookSecretHeader)
	if provided == "" || len(provided) != len(expected) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// ExtractMessage returns the most relevant message from an update.
func ExtractMessage(update *Update) *Message {
	if update == nil {
		return nil
	}
	if update.Message != nil {
		return update.Message
	}
	if update.EditedMessage != nil {
		return update.EditedMessage
	}
	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		return update.CallbackQuery.Message
	}
	return nil
}
