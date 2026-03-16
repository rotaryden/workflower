package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"workflower/config"
	"workflower/lib/telegram"
	"workflower/storage"
	"workflower/templates/ui_templates"
	"workflower/workflow"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	cfg       *config.Config
	store     *storage.Store
	engine    *workflow.Engine
	notifier  *telegram.Notifier
	templates *ui_templates.TemplatesList
}

// NewHandler creates a new handler instance
func NewHandler(cfg *config.Config, store *storage.Store, engine *workflow.Engine, templates *ui_templates.TemplatesList) *Handler {
	return &Handler{
		cfg:       cfg,
		store:     store,
		engine:    engine,
		notifier:  telegram.NewNotifier(cfg.TelegramBotToken, cfg.TelegramChatID),
		templates: templates,
	}
}

// RegisterRoutes sets up all HTTP routes
func (h *Handler) RegisterRoutes(r *fiber.App) {
	// Static pages
	r.Get("/", h.StartPage)
	r.Get("/workflows", h.WorkflowsList)
	r.Get("/workflow/:id", h.WorkflowStatus)
	r.Get("/review/:id", h.ReviewPage)

	// API endpoints
	r.Post("/workflow/start", h.StartWorkflow)
	r.Post("/workflow/:id/submit", h.SubmitReview)

	// Telegram webhook
	r.Post(normalizeWebhookPath(h.cfg.TelegramWebhookPath), h.TelegramWebhook)

	// Health check
	r.Get("/health", h.HealthCheck)
}

// StartPage renders the workflow starter form
func (h *Handler) StartPage(c *fiber.Ctx) error {
	data := ui_templates.PageData{
		Title: "Create Song",
	}

	var buf bytes.Buffer
	if err := h.templates.Start.Execute(&buf, data); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(fmt.Sprintf("Template error: %v", err))
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(buf.Bytes())
}

// WorkflowsList shows all workflows
func (h *Handler) WorkflowsList(c *fiber.Ctx) error {
	workflows := h.store.List()

	data := ui_templates.PageData{
		Title:     "Workflows",
		Workflows: workflows,
	}

	var buf bytes.Buffer
	if err := h.templates.List.Execute(&buf, data); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(fmt.Sprintf("Template error: %v", err))
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(buf.Bytes())
}

// WorkflowStatus shows the status of a specific workflow
func (h *Handler) WorkflowStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	wf, ok := h.store.Get(id)
	if !ok {
		return c.Status(http.StatusNotFound).SendString("Workflow not found")
	}

	// If awaiting review, redirect to review page
	if wf.Status == "awaiting_review" {
		return c.Redirect("/review/"+id, http.StatusFound)
	}

	data := ui_templates.PageData{
		Title:    "Workflow Status",
		Workflow: wf,
	}

	var buf bytes.Buffer
	if err := h.templates.Status.Execute(&buf, data); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(fmt.Sprintf("Template error: %v", err))
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(buf.Bytes())
}

// ReviewPage shows the human-in-the-loop review form
func (h *Handler) ReviewPage(c *fiber.Ctx) error {
	id := c.Params("id")

	wf, ok := h.store.Get(id)
	if !ok {
		return c.Status(http.StatusNotFound).SendString("Workflow not found")
	}

	if wf.Status != "awaiting_review" {
		return c.Redirect("/workflow/"+id, http.StatusFound)
	}

	data := ui_templates.PageData{
		Title:    "Review",
		Workflow: wf,
	}

	var buf bytes.Buffer
	if err := h.templates.Review.Execute(&buf, data); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(fmt.Sprintf("Template error: %v", err))
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(buf.Bytes())
}

// StartWorkflow handles the workflow creation request
func (h *Handler) StartWorkflow(c *fiber.Ctx) error {
	taskDescription := c.FormValue("task_description")
	if taskDescription == "" {
		return c.Status(http.StatusBadRequest).SendString("Task description is required")
	}

	isPremium := c.FormValue("is_premium") == "true"

	// Handle audio file upload
	var audioFilePath, audioFileName string
	fileHeader, err := c.FormFile("audio_file")
	if err == nil && fileHeader != nil {
		file, err := fileHeader.Open()
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString(fmt.Sprintf("Failed to open uploaded file: %v", err))
		}
		defer file.Close() //nolint:errcheck

		// Create uploads directory
		uploadsDir := filepath.Join("uploads", time.Now().Format("2006-01-02"))
		if err := os.MkdirAll(uploadsDir, 0755); err != nil {
			return c.Status(http.StatusInternalServerError).SendString(fmt.Sprintf("Failed to create uploads directory: %v", err))
		}

		// Save file
		audioFileName = fileHeader.Filename
		audioFilePath = filepath.Join(uploadsDir, uuid.New().String()+"_"+fileHeader.Filename)

		dst, err := os.Create(audioFilePath)
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString(fmt.Sprintf("Failed to save file: %v", err))
		}
		defer dst.Close() //nolint:errcheck

		if _, err := io.Copy(dst, file); err != nil {
			return c.Status(http.StatusInternalServerError).SendString(fmt.Sprintf("Failed to save file: %v", err))
		}
	}

	// Start the workflow
	ctx := context.Background()
	state, err := h.engine.StartWorkflow(ctx, taskDescription, isPremium, audioFilePath, audioFileName)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(fmt.Sprintf("Failed to start workflow: %v", err))
	}

	// Redirect to workflow status page
	return c.Redirect("/workflow/"+state.ID, http.StatusFound)
}

// SubmitReview handles the review form submission
func (h *Handler) SubmitReview(c *fiber.Ctx) error {
	id := c.Params("id")

	wf, ok := h.store.Get(id)
	if !ok {
		return c.Status(http.StatusNotFound).SendString("Workflow not found")
	}

	if wf.Status != "awaiting_review" {
		return c.Status(http.StatusBadRequest).SendString("Workflow is not awaiting review")
	}

	action := c.FormValue("action")

	if action == "reject" {
		h.engine.RejectWorkflow(wf)
		return c.Redirect("/workflow/"+id, http.StatusFound)
	}

	// Update with edited values
	wf.EditedLyrics = c.FormValue("edited_lyrics")

	// Parse properties
	weirdness, _ := strconv.ParseFloat(c.FormValue("weirdness"), 64)
	wf.EditedProperties = &storage.SunoProperties{
		Style:          c.FormValue("style"),
		VocalType:      c.FormValue("vocal_type"),
		Weirdness:      weirdness,
		StyleInfluence: c.FormValue("style_influence"),
	}

	// Update premium features if present
	if wf.IsPremium {
		persona := c.FormValue("persona")
		inspo := c.FormValue("inspo")
		if persona != "" || inspo != "" {
			wf.PersonaInspo = &storage.PersonaInspo{
				Persona: persona,
				Inspo:   inspo,
			}
		}
	}

	h.store.Save(wf)

	// Approve and submit to Suno
	ctx := context.Background()
	if err := h.engine.ApproveWorkflow(ctx, wf); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(fmt.Sprintf("Failed to approve workflow: %v", err))
	}

	return c.Redirect("/workflow/"+id, http.StatusFound)
}

// TelegramWebhook handles incoming Telegram webhook updates.
func (h *Handler) TelegramWebhook(c *fiber.Ctx) error {
	if h.cfg.TelegramBotToken == "" {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"status": "telegram_disabled"})
	}

	if !telegram.VerifyWebhookSecret(c.Get(telegram.WebhookSecretHeader), h.cfg.TelegramWebhookSecret) {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"status": "unauthorized"})
	}

	var update telegram.Update
	if err := c.BodyParser(&update); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"status": "invalid_payload"})
	}

	go h.handleTelegramUpdate(update)
	return c.SendStatus(http.StatusOK)
}

func (h *Handler) handleTelegramUpdate(update telegram.Update) {
	message := telegram.ExtractMessage(&update)
	if message == nil {
		return
	}
	if message.From != nil && message.From.IsBot {
		return
	}

	text := strings.TrimSpace(message.Text)
	if text == "" {
		text = strings.TrimSpace(message.Caption)
	}
	if text == "" {
		return
	}

	chatID := strconv.FormatInt(message.Chat.ID, 10)
	if h.cfg.TelegramChatID != "" && chatID != h.cfg.TelegramChatID {
		slog.Info("Telegram webhook ignored chat", "chat_id", chatID, "expected", h.cfg.TelegramChatID)
		return
	}

	baseURL := strings.TrimRight(h.cfg.BaseURL, "/")
	command, args := parseTelegramCommand(text)
	switch command {
	case "/start", "/help":
		h.replyTelegramHelp(chatID)
		return
	case "/status":
		if strings.TrimSpace(args) == "" {
			h.replyTelegramText(chatID, "Usage: /status WORKFLOW_ID")
			return
		}
		h.replyTelegramStatus(chatID, args, baseURL)
		return
	case "/premium":
		if strings.TrimSpace(args) == "" {
			h.replyTelegramText(chatID, "Usage: /premium your task description")
			return
		}
		h.startWorkflowFromTelegram(chatID, args, true, baseURL)
		return
	case "/basic":
		if strings.TrimSpace(args) == "" {
			h.replyTelegramText(chatID, "Usage: /basic your task description")
			return
		}
		h.startWorkflowFromTelegram(chatID, args, false, baseURL)
		return
	default:
		if command != "" {
			h.replyTelegramText(chatID, "Unknown command. Send /help for options.")
			return
		}
		h.startWorkflowFromTelegram(chatID, args, h.cfg.EnablePremiumFeatures, baseURL)
	}
}

func (h *Handler) startWorkflowFromTelegram(chatID, task string, isPremium bool, baseURL string) {
	task = strings.TrimSpace(task)
	if task == "" {
		h.replyTelegramText(chatID, "Task description is required.")
		return
	}

	ctx := context.Background()
	state, err := h.engine.StartWorkflow(ctx, task, isPremium, "", "")
	if err != nil {
		h.replyTelegramText(chatID, fmt.Sprintf("Failed to start workflow: %v", err))
		return
	}

	statusURL := fmt.Sprintf("%s/workflow/%s", baseURL, state.ID)
	reply := fmt.Sprintf("Workflow started.\n\nID: %s\nStatus: %s\nLink: %s", state.ID, state.Status, statusURL)
	h.replyTelegramText(chatID, reply)
}

func (h *Handler) replyTelegramStatus(chatID, workflowID, baseURL string) {
	id := strings.TrimSpace(workflowID)
	if id == "" {
		h.replyTelegramText(chatID, "Usage: /status WORKFLOW_ID")
		return
	}

	wf, ok := h.store.Get(id)
	if !ok {
		h.replyTelegramText(chatID, "Workflow not found.")
		return
	}

	statusURL := fmt.Sprintf("%s/workflow/%s", baseURL, wf.ID)
	reply := fmt.Sprintf("Status: %s\nLink: %s", wf.Status, statusURL)
	if wf.Status == "awaiting_review" {
		reviewURL := fmt.Sprintf("%s/review/%s", baseURL, wf.ID)
		reply = fmt.Sprintf("%s\nReview: %s", reply, reviewURL)
	}

	h.replyTelegramText(chatID, reply)
}

func (h *Handler) replyTelegramHelp(chatID string) {
	defaultMode := "basic"
	if h.cfg.EnablePremiumFeatures {
		defaultMode = "premium"
	}

	reply := fmt.Sprintf(
		"Send a task description to start a workflow.\nDefault mode: %s.\n\nCommands:\n/premium your task description\n/basic your task description\n/status WORKFLOW_ID",
		defaultMode,
	)
	h.replyTelegramText(chatID, reply)
}

func (h *Handler) replyTelegramText(chatID, message string) {
	if err := h.notifier.SendToChat(context.Background(), chatID, message); err != nil {
		slog.Warn("Failed to send Telegram reply", "error", err, "chat_id", chatID)
	}
}

// HealthCheck returns server health status
func (h *Handler) HealthCheck(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	})
}

// ErrorHandler is a middleware for handling panics
func ErrorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				_ = c.Status(http.StatusInternalServerError).SendString(fmt.Sprintf("Internal server error: %v", r))
			}
		}()
		return c.Next()
	}
}

func normalizeWebhookPath(path string) string {
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		return "/telegram/webhook"
	}
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	return normalized
}

func parseTelegramCommand(text string) (string, string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "", trimmed
	}

	parts := strings.Fields(trimmed)
	command := parts[0]
	if at := strings.Index(command, "@"); at != -1 {
		command = command[:at]
	}

	args := strings.TrimSpace(strings.TrimPrefix(trimmed, parts[0]))
	return strings.ToLower(command), args
}
