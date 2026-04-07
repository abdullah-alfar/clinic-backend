package followup

import (
	"encoding/json"
	"net/http"
	"strings"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	var req CreateFollowUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", nil)
		return
	}

	f, err := h.svc.CreateFollowUp(userCtx.TenantID, userCtx.UserID, req)
	if err != nil {
		if err == ErrInvalidDueDate {
			myhttp.RespondError(w, http.StatusBadRequest, err.Error(), "BAD_REQUEST", nil)
		} else {
			myhttp.RespondError(w, http.StatusInternalServerError, "failed to create follow-up", "INTERNAL_ERROR", nil)
		}
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, f, "follow-up created successfully")
}

func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	filters := ListFilters{
		Status:   r.URL.Query().Get("status"),
		Overdue:  r.URL.Query().Get("overdue") == "true",
		DueToday: r.URL.Query().Get("due_today") == "true",
	}

	if drID := r.URL.Query().Get("doctor_id"); drID != "" {
		id, _ := uuid.Parse(drID)
		if id != uuid.Nil {
			filters.DoctorID = &id
		}
	}

	followups, err := h.svc.ListFollowUps(userCtx.TenantID, filters)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to list follow-ups", "INTERNAL_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, followups, "success")
}

func (h *Handler) HandlePatientFollowUps(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	// /api/v1/patients/{id}/follow-ups
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid path", "BAD_REQUEST", nil)
		return
	}
	patientID, err := uuid.Parse(parts[4])
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient id", "BAD_REQUEST", nil)
		return
	}

	followups, err := h.svc.GetPatientFollowUps(userCtx.TenantID, patientID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch follow-ups", "INTERNAL_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, followups, "success")
}

func (h *Handler) HandleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid path", "BAD_REQUEST", nil)
		return
	}
	id, err := uuid.Parse(parts[4])
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid follow-up id", "BAD_REQUEST", nil)
		return
	}

	var req UpdateFollowUpStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", nil)
		return
	}

	err = h.svc.UpdateStatus(userCtx.TenantID, id, req.Status)
	if err != nil {
		if err == ErrNotFound {
			myhttp.RespondError(w, http.StatusNotFound, "not found", "NOT_FOUND", nil)
		} else {
			myhttp.RespondError(w, http.StatusInternalServerError, "update failed", "INTERNAL_ERROR", nil)
		}
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "status updated successfully")
}
