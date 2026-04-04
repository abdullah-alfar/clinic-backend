package recurrence

import (
	"encoding/json"
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"

	"github.com/google/uuid"
)

type RecurrenceHandler struct {
	service *RecurrenceService
}

func NewRecurrenceHandler(service *RecurrenceService) *RecurrenceHandler {
	return &RecurrenceHandler{service: service}
}

func (h *RecurrenceHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	uctx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "missing user context", "UNAUTHORIZED", nil)
		return
	}

	var req CreateRecurrenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", nil)
		return
	}

	rule, apptIDs, err := h.service.CreateRecurringAppointment(r.Context(), uctx.TenantID, uctx.UserID, req)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to create recurrence", "INTERNAL_ERROR", err)
		return
	}

	resp := map[string]any{
		"data": rule,
		"meta": map[string]any{
			"appointments_created": len(apptIDs),
		},
	}

	myhttp.RespondJSON(w, http.StatusCreated, resp, "recurrence created successfully")
}

func (h *RecurrenceHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	uctx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "missing user context", "UNAUTHORIZED", nil)
		return
	}

	patientIDStr := r.URL.Query().Get("patient_id")
	patientID, _ := uuid.Parse(patientIDStr)
	if patientID == uuid.Nil {
		myhttp.RespondError(w, http.StatusBadRequest, "missing patient_id", "BAD_REQUEST", nil)
		return
	}

	rules, err := h.service.repo.GetRulesByPatient(r.Context(), uctx.TenantID, patientID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch rules", "INTERNAL_ERROR", err)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, rules, "success")
}
