package ai_core

import (
	"encoding/json"
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
	"github.com/google/uuid"
)

type AIHandler struct {
	svc AIService
}

func NewAIHandler(svc AIService) *AIHandler {
	return &AIHandler{svc: svc}
}

// ChatRequest defines the payload structure expected from the Frontend Chat Panel or Global Search.
type ChatRequest struct {
	SessionID string                 `json:"session_id"` // Provided by frontend for continuing transient chats
	Input     string                 `json:"input"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Source    string                 `json:"source"`
	PatientID *uuid.UUID             `json:"patient_id,omitempty"` // If user is currently looking at a specific patient globally
}

// HandleChat provides HTTP POST /api/v1/ai/chat.
func (h *AIHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", nil)
		return
	}

	if req.Input == "" {
		myhttp.RespondError(w, http.StatusBadRequest, "input string cannot be empty", "BAD_REQUEST", nil)
		return
	}

	if req.SessionID == "" {
		// Start a new logical transient tracking identifier if omitted
		req.SessionID = uuid.NewString()
	}

	// Route into Orchestrator
	aiReq := AIRequest{
		SessionID: req.SessionID,
		TenantID:  userCtx.TenantID,
		UserID:    &userCtx.UserID,
		PatientID: req.PatientID,
		Input:     req.Input,
		Context:   req.Context,
		Source:    req.Source,
	}

	if aiReq.Source == "" {
		aiReq.Source = "web_chat"
	}

	resp, err := h.svc.Process(r.Context(), aiReq)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, err.Error(), "AI_PROCESSING_ERROR", nil)
		return
	}

	// Always append the resolved SessionID back so frontend can keep state attached
	responseWrapper := struct {
		SessionID string      `json:"session_id"`
		Message   string      `json:"message"`
		Action    string      `json:"action,omitempty"`
		Data      interface{} `json:"data,omitempty"`
	}{
		SessionID: req.SessionID,
		Message:   resp.Message,
		Action:    resp.Action,
		Data:      resp.Data,
	}

	myhttp.RespondJSON(w, http.StatusOK, responseWrapper, "success")
}
