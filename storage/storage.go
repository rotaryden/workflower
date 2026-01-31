package storage

import (
	"sync"
	"time"
)

// WorkflowState represents the state of a workflow instance
type WorkflowState struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Status    string    `json:"status"` // pending, awaiting_review, approved, rejected, completed, failed

	// Input
	TaskDescription string `json:"task_description"`
	IsPremium       bool   `json:"is_premium"`
	AudioFilePath   string `json:"audio_file_path,omitempty"`
	AudioFileName   string `json:"audio_file_name,omitempty"`

	// Generated content
	Lyrics              string `json:"lyrics,omitempty"`
	LyricsWithBrackets  string `json:"lyrics_with_brackets,omitempty"`
	SunoProperties      *SunoProperties `json:"suno_properties,omitempty"`
	PersonaInspo        *PersonaInspo   `json:"persona_inspo,omitempty"`

	// Human-in-the-loop edits
	EditedLyrics       string          `json:"edited_lyrics,omitempty"`
	EditedProperties   *SunoProperties `json:"edited_properties,omitempty"`

	// Suno result
	SunoJobID  string `json:"suno_job_id,omitempty"`
	SunoResult string `json:"suno_result,omitempty"`
	ErrorMsg   string `json:"error_msg,omitempty"`
}

// SunoProperties holds the Suno configuration
type SunoProperties struct {
	Style          string  `json:"style"`
	VocalType      string  `json:"vocal_type"`
	LyricsMode     string  `json:"lyrics_mode"`
	Weirdness      float64 `json:"weirdness"`
	StyleInfluence string  `json:"style_influence"`
}

// PersonaInspo holds premium Suno features
type PersonaInspo struct {
	Persona string `json:"persona"`
	Inspo   string `json:"inspo"`
}

// Store provides thread-safe in-memory storage for workflow states
type Store struct {
	mu        sync.RWMutex
	workflows map[string]*WorkflowState
}

// NewStore creates a new in-memory store
func NewStore() *Store {
	return &Store{
		workflows: make(map[string]*WorkflowState),
	}
}

// Save stores or updates a workflow state
func (s *Store) Save(state *WorkflowState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state.UpdatedAt = time.Now()
	s.workflows[state.ID] = state
}

// Get retrieves a workflow state by ID
func (s *Store) Get(id string) (*WorkflowState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.workflows[id]
	return state, ok
}

// Delete removes a workflow state
func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.workflows, id)
}

// List returns all workflow states
func (s *Store) List() []*WorkflowState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make([]*WorkflowState, 0, len(s.workflows))
	for _, state := range s.workflows {
		result = append(result, state)
	}
	return result
}

// ListByStatus returns workflow states with a specific status
func (s *Store) ListByStatus(status string) []*WorkflowState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []*WorkflowState
	for _, state := range s.workflows {
		if state.Status == status {
			result = append(result, state)
		}
	}
	return result
}

