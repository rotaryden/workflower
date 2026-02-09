package llm

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	promptsOnce sync.Once
	prompts     map[string]string
	promptsErr  error
)

// LoadPrompts loads all prompt templates from the templates/prompts directory
// If baseDir is empty, it uses the current working directory
func LoadPrompts(baseDir string) error {
	promptsOnce.Do(func() {
		prompts = make(map[string]string)
		
		if baseDir == "" {
			var err error
			baseDir, err = os.Getwd()
			if err != nil {
				promptsErr = fmt.Errorf("failed to get working directory: %w", err)
				return
			}
		}
		
		promptsDir := filepath.Join(baseDir, "templates", "prompts")

		// Define prompt files to load
		promptFiles := map[string]string{
			"lyrics_generation":    "lyrics_generation.txt",
			"suno_properties":       "suno_properties.txt",
			"bracket_instructions": "bracket_instructions.txt",
			"persona_inspo":        "persona_inspo.txt",
		}

		for key, filename := range promptFiles {
			filePath := filepath.Join(promptsDir, filename)
			content, err := os.ReadFile(filePath)
			if err != nil {
				promptsErr = fmt.Errorf("failed to load prompt %s from %s: %w", key, filePath, err)
				return
			}
			prompts[key] = string(content)
		}
	})

	return promptsErr
}

// GetPrompt retrieves a prompt template by name
func GetPrompt(name string) (string, error) {
	if prompts == nil {
		return "", fmt.Errorf("prompts not loaded, call LoadPrompts first")
	}

	prompt, exists := prompts[name]
	if !exists {
		return "", fmt.Errorf("prompt %s not found", name)
	}

	return prompt, nil
}

// LyricsGenerationPrompt returns the lyrics generation prompt template
func LyricsGenerationPrompt() (string, error) {
	return GetPrompt("lyrics_generation")
}

// SunoPropertiesPrompt returns the Suno properties prompt template
func SunoPropertiesPrompt() (string, error) {
	return GetPrompt("suno_properties")
}

// BracketInstructionsPrompt returns the bracket instructions prompt template
func BracketInstructionsPrompt() (string, error) {
	return GetPrompt("bracket_instructions")
}

// PersonaInspoPrompt returns the persona/inspo prompt template
func PersonaInspoPrompt() (string, error) {
	return GetPrompt("persona_inspo")
}
