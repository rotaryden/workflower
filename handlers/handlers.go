package handlers

import (
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

	"github.com/gin-gonic/gin"
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
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Static pages
	r.GET("/", h.StartPage)
	r.GET("/workflows", h.WorkflowsList)
	r.GET("/workflow/:id", h.WorkflowStatus)
	r.GET("/review/:id", h.ReviewPage)

	// API endpoints
	r.POST("/workflow/start", h.StartWorkflow)
	r.POST("/workflow/:id/submit", h.SubmitReview)

	// Telegram webhook
	r.POST(normalizeWebhookPath(h.cfg.TelegramWebhookPath), h.TelegramWebhook)

	// Health check
	r.GET("/health", h.HealthCheck)
}

// StartPage renders the workflow starter form
func (h *Handler) StartPage(c *gin.Context) {
	data := ui_templates.PageData{
		Title: "Create Song",
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.Start.Execute(c.Writer, data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

// WorkflowsList shows all workflows
func (h *Handler) WorkflowsList(c *gin.Context) {
	workflows := h.store.List()

	data := ui_templates.PageData{
		Title:     "Workflows",
		Workflows: workflows,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.List.Execute(c.Writer, data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

// WorkflowStatus shows the status of a specific workflow
func (h *Handler) WorkflowStatus(c *gin.Context) {
	id := c.Param("id")

	wf, ok := h.store.Get(id)
	if !ok {
		c.String(http.StatusNotFound, "Workflow not found")
		return
	}

	// If awaiting review, redirect to review page
	if wf.Status == "awaiting_review" {
		c.Redirect(http.StatusFound, "/review/"+id)
		return
	}

	data := ui_templates.PageData{
		Title:    "Workflow Status",
		Workflow: wf,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.Status.Execute(c.Writer, data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

// ReviewPage shows the human-in-the-loop review form
func (h *Handler) ReviewPage(c *gin.Context) {
	id := c.Param("id")

	wf, ok := h.store.Get(id)
	if !ok {
		c.String(http.StatusNotFound, "Workflow not found")
		return
	}

	if wf.Status != "awaiting_review" {
		c.Redirect(http.StatusFound, "/workflow/"+id)
		return
	}

	data := ui_templates.PageData{
		Title:    "Review",
		Workflow: wf,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.Review.Execute(c.Writer, data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

// StartWorkflow handles the workflow creation request
func (h *Handler) StartWorkflow(c *gin.Context) {
	// Parse form data
	if err := c.Request.ParseMultipartForm(int64(h.cfg.MaxAudioSizeMB) << 20); err != nil {
		// Try regular form parsing
		if err := c.Request.ParseForm(); err != nil {
			c.String(http.StatusBadRequest, "Failed to parse form: %v", err)
			return
		}
	}

	taskDescription := c.PostForm("task_description")
	if taskDescription == "" {
		c.String(http.StatusBadRequest, "Task description is required")
		return
	}

	isPremium := c.PostForm("is_premium") == "true"

	// Handle audio file upload
	var audioFilePath, audioFileName string
	file, header, err := c.Request.FormFile("audio_file")
	if err == nil && file != nil {
		defer file.Close()

		// Create uploads directory
		uploadsDir := filepath.Join("uploads", time.Now().Format("2006-01-02"))
		if err := os.MkdirAll(uploadsDir, 0755); err != nil {
			c.String(http.StatusInternalServerError, "Failed to create uploads directory: %v", err)
			return
		}

		// Save file
		audioFileName = header.Filename
		audioFilePath = filepath.Join(uploadsDir, uuid.New().String()+"_"+header.Filename)

		dst, err := os.Create(audioFilePath)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to save file: %v", err)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			c.String(http.StatusInternalServerError, "Failed to save file: %v", err)
			return
		}
	}

	// Start the workflow
	ctx := context.Background()
	state, err := h.engine.StartWorkflow(ctx, taskDescription, isPremium, audioFilePath, audioFileName)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to start workflow: %v", err)
		return
	}

	// Redirect to workflow status page
	c.Redirect(http.StatusFound, "/workflow/"+state.ID)
}

// SubmitReview handles the review form submission
func (h *Handler) SubmitReview(c *gin.Context) {
	id := c.Param("id")

	wf, ok := h.store.Get(id)
	if !ok {
		c.String(http.StatusNotFound, "Workflow not found")
		return
	}

	if wf.Status != "awaiting_review" {
		c.String(http.StatusBadRequest, "Workflow is not awaiting review")
		return
	}

	action := c.PostForm("action")

	if action == "reject" {
		h.engine.RejectWorkflow(wf)
		c.Redirect(http.StatusFound, "/workflow/"+id)
		return
	}

	// Update with edited values
	wf.EditedLyrics = c.PostForm("edited_lyrics")

	// Parse properties
	weirdness, _ := strconv.ParseFloat(c.PostForm("weirdness"), 64)
	wf.EditedProperties = &storage.SunoProperties{
		Style:          c.PostForm("style"),
		VocalType:      c.PostForm("vocal_type"),
		Weirdness:      weirdness,
		StyleInfluence: c.PostForm("style_influence"),
	}

	// Update premium features if present
	if wf.IsPremium {
		persona := c.PostForm("persona")
		inspo := c.PostForm("inspo")
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
		c.String(http.StatusInternalServerError, "Failed to approve workflow: %v", err)
		return
	}

	c.Redirect(http.StatusFound, "/workflow/"+id)
}

// TelegramWebhook handles incoming Telegram webhook updates.
func (h *Handler) TelegramWebhook(c *gin.Context) {
	if h.cfg.TelegramBotToken == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "telegram_disabled"})
		return
	}

	if !telegram.VerifyWebhookSecret(c.Request, h.cfg.TelegramWebhookSecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
		return
	}

	var update telegram.Update
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "invalid_payload"})
		return
	}

	c.Status(http.StatusOK)
	go h.handleTelegramUpdate(update)
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
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	})
}

// ErrorHandler is a middleware for handling panics
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.String(http.StatusInternalServerError, fmt.Sprintf("Internal server error: %v", err))
				c.Abort()
			}
		}()
		c.Next()
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
