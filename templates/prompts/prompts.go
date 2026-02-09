package prompts

import (
	_ "embed"
)

// Embed prompt templates at compile time
//
//go:embed lyrics_generation.txt
var lyricsGenerationPrompt string

//go:embed suno_properties.txt
var sunoPropertiesPrompt string

//go:embed bracket_instructions.txt
var bracketInstructionsPrompt string

//go:embed persona_inspo.txt
var personaInspoPrompt string

type PromptsList struct {
	LyricsGeneration    string
	SunoProperties      string
	BracketInstructions string
	PersonaInspo        string
}

// Init initializes the prompts list with embedded content
func Init() *PromptsList {
	return &PromptsList{
		LyricsGeneration:    lyricsGenerationPrompt,
		SunoProperties:      sunoPropertiesPrompt,
		BracketInstructions: bracketInstructionsPrompt,
		PersonaInspo:        personaInspoPrompt,
	}
}
