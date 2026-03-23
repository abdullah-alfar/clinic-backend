package appointment

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"

	"github.com/google/uuid"
)

type AppointmentHandler struct {
	svc *AppointmentService
}

func NewAppointmentHandler(svc *AppointmentService) *AppointmentHandler {
	return &AppointmentHandler{svc: svc}
}

type ScheduleRequest struct {
	PatientID uuid.UUID `json:"patient_id"`
	DoctorID  uuid.UUID `json:"doctor_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

func (h *AppointmentHandler) HandleSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	var req ScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", err.Error())
		return
	}

	appt, err := h.svc.ScheduleAppointment(userCtx.TenantID, req.PatientID, req.DoctorID, req.StartTime, req.EndTime, userCtx.UserID)
	if err != nil {
		if err == ErrDoubleBooking || err == ErrDoctorInactive || err == ErrInvalidTime || err == ErrPastAppointment {
			myhttp.RespondError(w, http.StatusConflict, err.Error(), "CONFLICT", err.Error())
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, appt, "appointment scheduled successfully")
}

type UpdateTimeRequest struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

func (h *AppointmentHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	// Extract ID from path e.g. /api/v1/appointments/{id}
	// For simplicity, assuming exact matching or using a router
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid url", "BAD_REQUEST", nil)
		return
	}
	apptIDStr := pathParts[4]
	apptID, err := uuid.Parse(apptIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid appointment id", "BAD_REQUEST", nil)
		return
	}

	var req UpdateTimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", nil)
		return
	}

	if err := h.svc.UpdateAppointmentTime(userCtx.TenantID, apptID, req.StartTime, req.EndTime, userCtx.UserID); err != nil {
		myhttp.RespondError(w, http.StatusConflict, err.Error(), "CONFLICT", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "appointment updated successfully")
}

func (h *AppointmentHandler) HandleStatus(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
			return
		}

		userCtx, ok := shared.GetUserContext(r.Context())
		if !ok {
			myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
			return
		}

		// URL: /api/v1/appointments/{id}/statusName
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 6 {
			myhttp.RespondError(w, http.StatusBadRequest, "invalid url", "BAD_REQUEST", nil)
			return
		}
		apptIDStr := pathParts[4]
		apptID, err := uuid.Parse(apptIDStr)
		if err != nil {
			myhttp.RespondError(w, http.StatusBadRequest, "invalid appointment id", "BAD_REQUEST", nil)
			return
		}

		if err := h.svc.UpdateStatus(userCtx.TenantID, apptID, status, userCtx.UserID); err != nil {
			myhttp.RespondError(w, http.StatusConflict, err.Error(), "CONFLICT", nil)
			return
		}
		myhttp.RespondJSON(w, http.StatusOK, nil, "appointment status updated to "+status)
	}
}
