package ai_agent

import (
	"encoding/json"
	"time"
)

type AIRequest struct {
	TenantID  string         `json:"tenant_id"`
	PatientID string         `json:"patient_id,omitempty"`
	SessionID string         `json:"session_id"`
	Input     string         `json:"input"`
	Context   map[string]any `json:"context,omitempty"`
}

type AIResponse struct {
	Message              string          `json:"message"`
	RequiresConfirmation bool            `json:"requires_confirmation"`
	Action               *PendingAction  `json:"action,omitempty"`
}

type PendingAction struct {
	Token       string          `json:"token"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Summary     string          `json:"summary"`
	CreatedAt   time.Time       `json:"created_at"`
	Confirmed   bool            `json:"confirmed"`
}

type ReactObservation struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ReactResult struct {
	Thought     string          `json:"thought"`
	Action      string          `json:"action"`
	Input       json.RawMessage `json:"input"`
	FinalAnswer string          `json:"final_answer"`
}
