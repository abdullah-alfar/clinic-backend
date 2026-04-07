package settings

import (
	"encoding/json"
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
)

// Handler serves all settings API endpoints.
type Handler struct {
	svc *Service
}

// NewHandler returns a new settings Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// HandleGetSettings serves GET /api/v1/settings
func (h *Handler) HandleGetSettings(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	resp, err := h.svc.GetSettings(userCtx.TenantID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to load settings", "INTERNAL_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, resp, "success")
}

// HandleUpdateSettings serves PUT /api/v1/settings
func (h *Handler) HandleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", nil)
		return
	}

	if err := h.svc.UpdateSettings(userCtx.TenantID, req); err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to save settings", "INTERNAL_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "settings saved successfully")
}

// HandleTestAI serves POST /api/v1/settings/test-ai
func (h *Handler) HandleTestAI(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	var req TestAIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Prompt == "" {
		myhttp.RespondError(w, http.StatusBadRequest, "prompt is required", "BAD_REQUEST", nil)
		return
	}

	response, err := h.svc.TestAI(userCtx.TenantID, req.Prompt)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, err.Error(), "TEST_FAILED", nil)
		return
	}

	settings, _ := h.svc.GetSettings(userCtx.TenantID)
	provider := "log"
	if settings != nil {
		provider = settings.AIProvider
	}

	myhttp.RespondJSON(w, http.StatusOK, TestAIResponse{
		Response: response,
		Provider: provider,
	}, "success")
}

// HandleTestEmail serves POST /api/v1/settings/test-email
func (h *Handler) HandleTestEmail(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	var req TestEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.To == "" {
		myhttp.RespondError(w, http.StatusBadRequest, "recipient email (to) is required", "BAD_REQUEST", nil)
		return
	}

	if err := h.svc.TestEmail(userCtx.TenantID, req.To); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, err.Error(), "TEST_FAILED", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "test email sent successfully")
}

// HandleTestWhatsApp serves POST /api/v1/settings/test-whatsapp
func (h *Handler) HandleTestWhatsApp(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	var req TestWhatsAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.To == "" {
		myhttp.RespondError(w, http.StatusBadRequest, "recipient phone (to) is required", "BAD_REQUEST", nil)
		return
	}

	if err := h.svc.TestWhatsApp(userCtx.TenantID, req.To); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, err.Error(), "TEST_FAILED", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "test WhatsApp message sent successfully")
}
