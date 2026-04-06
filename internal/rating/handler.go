package rating

import (
	"encoding/json"
	"net/http"

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

func (h *Handler) HandleSubmitRating(w http.ResponseWriter, r *http.Request) {
	uctx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	apptIDStr := r.PathValue("id")
	apptID, err := uuid.Parse(apptIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid appointment id", "INVALID_ID", nil)
		return
	}

	var req CreateRatingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "INVALID_BODY", nil)
		return
	}

	rt, err := h.svc.SubmitRating(r.Context(), uctx.TenantID, uctx.UserID, apptID, req)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL_ERROR"
		if err == ErrAppointmentNotCompleted || err == ErrDuplicateRating || err == ErrInvalidRatingValue {
			status = http.StatusBadRequest
			code = "BAD_REQUEST"
		}
		myhttp.RespondError(w, status, err.Error(), code, nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, rt, "Rating submitted successfully")
}

func (h *Handler) HandleGetDoctorRatings(w http.ResponseWriter, r *http.Request) {
	uctx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	doctorIDStr := r.PathValue("id")
	doctorID, err := uuid.Parse(doctorIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusNotFound, "invalid doctor id", "INVALID_ID", nil)
		return
	}

	analytics, err := h.svc.GetDoctorFeed(r.Context(), uctx.TenantID, doctorID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to get ratings", "INTERNAL_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, analytics, "Doctor ratings retrieved")
}

func (h *Handler) HandleGetPatientRatings(w http.ResponseWriter, r *http.Request) {
	uctx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	patientIDStr := r.PathValue("id")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusNotFound, "invalid patient id", "INVALID_ID", nil)
		return
	}

	ratings, err := h.svc.GetPatientRatings(r.Context(), uctx.TenantID, patientID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to get ratings", "INTERNAL_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, ratings, "Patient ratings retrieved")
}

func (h *Handler) HandleGetGlobalAnalytics(w http.ResponseWriter, r *http.Request) {
	uctx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	analytics, err := h.svc.GetGlobalAnalytics(r.Context(), uctx.TenantID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to get analytics", "INTERNAL_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, analytics, "Global ratings analytics retrieved")
}
