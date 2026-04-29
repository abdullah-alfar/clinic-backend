package ai_agent

import (
	"encoding/json"
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
)

type AgentHandler struct {
	svc AgentService
}

func NewAgentHandler(svc AgentService) *AgentHandler {
	return &AgentHandler{svc: svc}
}

func (h *AgentHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
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
	defer r.Body.Close()

	if req.Message == "" {
		myhttp.RespondError(w, http.StatusBadRequest, "message is required", "BAD_REQUEST", nil)
		return
	}

	resp, err := h.svc.ProcessChat(r.Context(), req, userCtx.TenantID.String())
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, resp, "success")
}

func (h *AgentHandler) HandleConfirm(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	var req ConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", nil)
		return
	}
	defer r.Body.Close()

	if req.Token == "" {
		myhttp.RespondError(w, http.StatusBadRequest, "token is required", "BAD_REQUEST", nil)
		return
	}

	resp, err := h.svc.ProcessConfirmation(r.Context(), req, userCtx.TenantID.String())
	if err != nil {
		if err == ErrActionNotFound || err == ErrActionAlreadyDone {
			myhttp.RespondError(w, http.StatusBadRequest, err.Error(), "BAD_REQUEST", nil)
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, resp, "success")
}
