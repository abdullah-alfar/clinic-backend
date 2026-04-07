package patientprofile

import (
	"encoding/json"
	"fmt"
	"net/http"

	"clinic-backend/internal/shared"
	myhttp "clinic-backend/internal/platform/http"
	"github.com/google/uuid"
)

type PatientProfileHandler struct {
	service *PatientProfileService
}

func NewHandler(service *PatientProfileService) *PatientProfileHandler {
	return &PatientProfileHandler{service: service}
}

func (h *PatientProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	uctx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized: missing user context", "UNAUTHORIZED", nil)
		return
	}

	patientIDStr := r.PathValue("id")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient id", "BAD_REQUEST", nil)
		return
	}
    // Re-writing more cleanly
	h.handleGetProfile(w, r, uctx, patientID)
}

func (h *PatientProfileHandler) handleGetProfile(w http.ResponseWriter, r *http.Request, uctx *shared.UserContext, patientID uuid.UUID) {
	profile, err := h.service.GetPatientProfile(r.Context(), uctx.TenantID, patientID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PatientProfileResponse{
		Data:    *profile,
		Message: "success",
	})
}

func (h *PatientProfileHandler) GetActivityStream(w http.ResponseWriter, r *http.Request) {
	uctx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	patientIDStr := r.PathValue("id")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient id", "BAD_REQUEST", nil)
		return
	}

	h.handleGetActivityStream(w, r, uctx, patientID)
}

func (h *PatientProfileHandler) handleGetActivityStream(w http.ResponseWriter, r *http.Request, uctx *shared.UserContext, patientID uuid.UUID) {
	q := r.URL.Query()
	var (
		page  int
		limit int
	)

	if pStr := q.Get("page"); pStr != "" {
		fmt.Sscanf(pStr, "%d", &page)
	}
	if lStr := q.Get("limit"); lStr != "" {
		fmt.Sscanf(lStr, "%d", &limit)
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	stream, err := h.service.GetActivityStream(r.Context(), uctx.TenantID, patientID, page, limit)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stream)
}


