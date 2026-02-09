package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"workflower/config"
	"workflower/lib/llm"
	"workflower/lib/telegram"
	"workflower/storage"
	"workflower/suno"
	"workflower/templates/prompts"

	"github.com/google/uuid"
)

// Engine orchestrates the song creation workflow
type Engine struct {
	cfg         *config.Config
	llmClient   *llm.Client
	sunoAPI     *suno.Client
	notifier    *telegram.Notifier
	store       *storage.Store
	promptsList *prompts.PromptsList
}

// NewEngine creates a new workflow engine
func NewEngine(cfg *config.Config, store *storage.Store, promptsList *prompts.PromptsList) *Engine {
	return &Engine{
		cfg:         cfg,
		llmClient:   llm.NewClient(cfg.OpenAIAPIKey, cfg.OpenAIModel),
		sunoAPI:     suno.NewClient(cfg.SunoAPIKey, cfg.SunoBaseURL),
		notifier:    telegram.NewNotifier(cfg.TelegramBotToken, cfg.TelegramChatID),
		store:       store,
		promptsList: promptsList,
	}
}

// StartWorkflow begins a new song creation workflow
func (e *Engine) StartWorkflow(ctx context.Context, taskDescription string, isPremium bool, audioFilePath, audioFileName string) (*storage.WorkflowState, error) {
	// Create new workflow state
	state := &storage.WorkflowState{
		ID:              uuid.New().String(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Status:          "processing",
		TaskDescription: taskDescription,
		IsPremium:       isPremium,
		AudioFilePath:   audioFilePath,
		AudioFileName:   audioFileName,
	}
	e.store.Save(state)

	// Run the workflow steps asynchronously
	go e.runWorkflowSteps(ctx, state)

	return state, nil
}

// runWorkflowSteps executes all workflow steps
func (e *Engine) runWorkflowSteps(ctx context.Context, state *storage.WorkflowState) {
	var err error

	// Step 1: Generate lyrics
	state.Lyrics, err = e.generateLyrics(ctx, state.TaskDescription)
	if err != nil {
		e.handleError(state, "lyrics generation", err)
		return
	}
	e.store.Save(state)

	// Step 2: Determine Suno properties
	state.SunoProperties, err = e.determineSunoProperties(ctx, state.TaskDescription, state.Lyrics)
	if err != nil {
		e.handleError(state, "suno properties", err)
		return
	}
	e.store.Save(state)

	// Step 3: Add bracket instructions to lyrics
	state.LyricsWithBrackets, err = e.addBracketInstructions(ctx, state.Lyrics, state.SunoProperties)
	if err != nil {
		e.handleError(state, "bracket instructions", err)
		return
	}
	e.store.Save(state)

	// Step 4: Add Persona and Inspo (premium only)
	if state.IsPremium {
		state.PersonaInspo, err = e.generatePersonaInspo(ctx, state.TaskDescription, state.SunoProperties)
		if err != nil {
			e.handleError(state, "persona/inspo", err)
			return
		}
		e.store.Save(state)
	}

	// Step 5: Update status and notify for human review
	state.Status = "awaiting_review"
	state.EditedLyrics = state.LyricsWithBrackets
	state.EditedProperties = state.SunoProperties
	e.store.Save(state)

	// Notify via Telegram
	reviewURL := fmt.Sprintf("%s/review/%s", e.cfg.BaseURL, state.ID)
	message := fmt.Sprintf("🎵 Song workflow ready for review!\n\nTask: %s\n\n🔗 Review: %s",
		truncateString(state.TaskDescription, 100), reviewURL)

	if err := e.notifier.Send(ctx, message); err != nil {
		// Log but don't fail the workflow
		fmt.Printf("Warning: failed to send Telegram notification: %v\n", err)
	}
}

// generateLyrics creates song lyrics from the task description
func (e *Engine) generateLyrics(ctx context.Context, taskDescription string) (string, error) {
	return e.llmClient.Chat(ctx, e.promptsList.LyricsGeneration, taskDescription)
}

// determineSunoProperties generates optimal Suno configuration
func (e *Engine) determineSunoProperties(ctx context.Context, taskDescription, lyrics string) (*storage.SunoProperties, error) {
	userPrompt := fmt.Sprintf("Subject Description:\n%s\n\nLyrics:\n%s", taskDescription, lyrics)

	response, err := e.llmClient.Chat(ctx, e.promptsList.SunoProperties, userPrompt)
	if err != nil {
		return nil, err
	}

	var props storage.SunoProperties
	if err := json.Unmarshal([]byte(response), &props); err != nil {
		// Try to extract JSON from response if it contains extra text
		props, err = extractSunoProperties(response)
		if err != nil {
			return nil, fmt.Errorf("failed to parse suno properties: %w", err)
		}
	}

	return &props, nil
}

// addBracketInstructions enhances lyrics with Suno bracket instructions
func (e *Engine) addBracketInstructions(ctx context.Context, lyrics string, props *storage.SunoProperties) (string, error) {
	userPrompt := fmt.Sprintf("Original Lyrics:\n%s\n\nSong Style: %s\nVocal Type: %s",
		lyrics, props.Style, props.VocalType)

	return e.llmClient.Chat(ctx, e.promptsList.BracketInstructions, userPrompt)
}

// generatePersonaInspo creates premium Suno features
func (e *Engine) generatePersonaInspo(ctx context.Context, taskDescription string, props *storage.SunoProperties) (*storage.PersonaInspo, error) {
	userPrompt := fmt.Sprintf("Subject: %s\nStyle: %s\nVocal Type: %s",
		taskDescription, props.Style, props.VocalType)

	response, err := e.llmClient.Chat(ctx, e.promptsList.PersonaInspo, userPrompt)
	if err != nil {
		return nil, err
	}

	var pi storage.PersonaInspo
	if err := json.Unmarshal([]byte(response), &pi); err != nil {
		// Try to extract JSON from response
		pi, err = extractPersonaInspo(response)
		if err != nil {
			return nil, fmt.Errorf("failed to parse persona/inspo: %w", err)
		}
	}

	return &pi, nil
}

// ApproveWorkflow processes the approved workflow
func (e *Engine) ApproveWorkflow(ctx context.Context, state *storage.WorkflowState) error {
	state.Status = "approved"
	e.store.Save(state)

	// Submit to Suno
	go e.submitToSuno(ctx, state)

	return nil
}

// submitToSuno sends the song request to Suno API
func (e *Engine) submitToSuno(ctx context.Context, state *storage.WorkflowState) {
	props := state.EditedProperties
	if props == nil {
		props = state.SunoProperties
	}

	lyrics := state.EditedLyrics
	if lyrics == "" {
		lyrics = state.LyricsWithBrackets
	}

	req := &suno.GenerateRequest{
		Lyrics:         lyrics,
		Style:          props.Style,
		VocalType:      props.VocalType,
		Weirdness:      props.Weirdness,
		StyleInfluence: props.StyleInfluence,
	}

	// Add premium features if available
	if state.IsPremium && state.PersonaInspo != nil {
		req.Persona = state.PersonaInspo.Persona
		req.Inspo = state.PersonaInspo.Inspo
	}

	// Add audio reference if provided
	if state.AudioFilePath != "" {
		req.AudioReferencePath = state.AudioFilePath
	}

	result, err := e.sunoAPI.Generate(ctx, req)
	if err != nil {
		e.handleError(state, "suno submission", err)
		return
	}

	state.SunoJobID = result.JobID
	state.SunoResult = result.Status
	state.Status = "completed"
	e.store.Save(state)

	// Notify completion
	message := fmt.Sprintf("✅ Song generation completed!\n\nJob ID: %s\nStatus: %s", result.JobID, result.Status)
	if err := e.notifier.Send(ctx, message); err != nil {
		fmt.Printf("Warning: failed to send completion notification: %v\n", err)
	}
}

// RejectWorkflow marks the workflow as rejected
func (e *Engine) RejectWorkflow(state *storage.WorkflowState) {
	state.Status = "rejected"
	e.store.Save(state)
}

// handleError updates state with error information
func (e *Engine) handleError(state *storage.WorkflowState, step string, err error) {
	state.Status = "failed"
	state.ErrorMsg = fmt.Sprintf("%s failed: %v", step, err)
	e.store.Save(state)
	fmt.Printf("Workflow %s error at %s: %v\n", state.ID, step, err)
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func extractSunoProperties(response string) (storage.SunoProperties, error) {
	var props storage.SunoProperties

	// Try to find JSON in the response
	start := -1
	end := -1
	braceCount := 0

	for i, c := range response {
		if c == '{' {
			if start == -1 {
				start = i
			}
			braceCount++
		} else if c == '}' {
			braceCount--
			if braceCount == 0 && start != -1 {
				end = i + 1
				break
			}
		}
	}

	if start != -1 && end != -1 {
		if err := json.Unmarshal([]byte(response[start:end]), &props); err == nil {
			return props, nil
		}
	}

	return props, fmt.Errorf("no valid JSON found in response")
}

func extractPersonaInspo(response string) (storage.PersonaInspo, error) {
	var pi storage.PersonaInspo

	start := -1
	end := -1
	braceCount := 0

	for i, c := range response {
		if c == '{' {
			if start == -1 {
				start = i
			}
			braceCount++
		} else if c == '}' {
			braceCount--
			if braceCount == 0 && start != -1 {
				end = i + 1
				break
			}
		}
	}

	if start != -1 && end != -1 {
		if err := json.Unmarshal([]byte(response[start:end]), &pi); err == nil {
			return pi, nil
		}
	}

	return pi, fmt.Errorf("no valid JSON found in response")
}
