package whatsappbot

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// BotSession tracks the conversational state for a phone number.
type BotSession struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	PatientID     *uuid.UUID
	PhoneNumber   string
	CurrentFlow   string // e.g., "menu", "book", "cancel"
	CurrentStep   string // e.g., "start", "select_doctor", "confirm"
	State         StateData // generic JSON payload for temporary data
	LastMessageAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// StateData holds the payload inside the JSONB state column.
type StateData map[string]any

// Scan realizes StateData from the DB JSONB.
func (s *StateData) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, s)
}

// Set saves a key-value pair in state.
func (s StateData) Set(key string, val any) {
	s[key] = val
}

// Get reads a key from state.
func (s StateData) Get(key string) any {
	return s[key]
}

// Clear resets the conversational state but keeps basic cache.
func (s StateData) Clear() {
	for k := range s {
		delete(s, k)
	}
}
