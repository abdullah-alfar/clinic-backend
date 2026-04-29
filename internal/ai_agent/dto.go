package ai_agent

import "encoding/json"

type ChatRequest struct {
	Message   string                 `json:"message" validate:"required"`
	SessionID string                 `json:"session_id,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

type ChatResponse struct {
	Message              string         `json:"message"`
	RequiresConfirmation bool           `json:"requires_confirmation"`
	Action               *PendingAction `json:"action,omitempty"`
}

type ConfirmRequest struct {
	Token string `json:"token" validate:"required"`
}

type ConfirmResponse struct {
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result,omitempty"`
}
