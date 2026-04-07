package ai_core

import (
	"encoding/json"

	"github.com/google/uuid"
)

// AIRequest represents the unified input sent to the orchestrated AI Core pipeline.
// Source can be "bot" (WhatsApp), "search" (Global Bar), "report" (Medical Insights), etc.
type AIRequest struct {
	SessionID string                 `json:"session_id"` // Group conversations. Example: Pat_Phone, UserID_Browser
	TenantID  uuid.UUID              `json:"tenant_id"`
	UserID    *uuid.UUID             `json:"user_id,omitempty"` // For web UI calls
	PatientID *uuid.UUID             `json:"patient_id,omitempty"` // Contextual link to a specific patient
	Input     string                 `json:"input"`
	Context   map[string]interface{} `json:"context"` // e.g. current URL, page state, user role limiters
	Source    string                 `json:"source"`
}

// AIResponse is returned by the AI System. It will contain text explicitly meant for the user,
// and optionally an action the frontend/caller should take.
type AIResponse struct {
	Message string      `json:"message"` // The human readable response.
	Action  string      `json:"action,omitempty"`  // A structured command (e.g. "navigate", "refresh") 
	Data    interface{} `json:"data,omitempty"`    // Attached payload depending on Action
}

// ReactObservation defines an internal data structure used inside the orchestrator
// to keep a log of iterations (Thought -> Action -> Observation).
type ReactObservation struct {
	Role    string // "user", "assistant", "tool"
	Content string // The raw string data of the step
}

// ToolInvocation is matched against the parsed JSON when the LLM attempts to call a tool.
type ToolInvocation struct {
	Action string          `json:"action"` // The exact Name of the registered Tool.
	Input  json.RawMessage `json:"input"`  // JSON input parameters mapped to the tool.
}

// ReactResult represents the final parsed structure the AI replies with when it solves the prompt.
type ReactResult struct {
	FinalAnswer string `json:"final_answer,omitempty"`
	Action      string `json:"action,omitempty"`
	Input       json.RawMessage `json:"input,omitempty"`
}
