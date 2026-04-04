package notification

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"

	"github.com/google/uuid"
)

type NotificationHandler struct {
	svc   *NotificationService
	repo  NotificationRepository
	prefs *PreferenceService
}

func NewNotificationHandler(svc *NotificationService, repo NotificationRepository, prefs *PreferenceService) *NotificationHandler {
	return &NotificationHandler{
		svc:   svc,
		repo:  repo,
		prefs: prefs,
	}
}

func (h *NotificationHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	list, err := h.svc.List(userCtx.TenantID, userCtx.UserID, limit, offset)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch notifications", "DB_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, list, "success")
}

func (h *NotificationHandler) HandleRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid url", "BAD_REQUEST", nil)
		return
	}
	
	idStr := parts[4]
	id, err := uuid.Parse(idStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid notification id", "BAD_REQUEST", nil)
		return
	}

	if err := h.svc.MarkRead(userCtx.TenantID, userCtx.UserID, id); err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to mark as read", "DB_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "notification marked read")
}

func (h *NotificationHandler) HandlePatientHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	// /api/v1/patients/{id}/notifications
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid url", "BAD_REQUEST", nil)
		return
	}

	patientID, err := uuid.Parse(parts[4])
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient id", "BAD_REQUEST", nil)
		return
	}

	limit, offset := 50, 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	history, err := h.repo.ListByPatient(r.Context(), userCtx.TenantID, patientID, limit, offset)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch history", "DB_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, history, "success")
}

func (h *NotificationHandler) HandleGetPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	// /api/v1/patients/{id}/notification-preferences
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid url", "BAD_REQUEST", nil)
		return
	}

	patientID, err := uuid.Parse(parts[4])
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient id", "BAD_REQUEST", nil)
		return
	}

	prefs, err := h.prefs.GetPreferences(r.Context(), userCtx.TenantID, patientID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch preferences", "DB_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, prefs, "success")
}

func (h *NotificationHandler) HandleUpdatePreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid url", "BAD_REQUEST", nil)
		return
	}

	patientID, err := uuid.Parse(parts[4])
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient id", "BAD_REQUEST", nil)
		return
	}

	var req UpsertPreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", err.Error())
		return
	}

	prefs, err := h.prefs.UpsertPreferences(r.Context(), userCtx.TenantID, patientID, req)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to update preferences", "DB_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, prefs, "success")
}
