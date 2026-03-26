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
	svc      *AppointmentService
	availSvc *AvailabilityService
}

func NewAppointmentHandler(svc *AppointmentService, availSvc *AvailabilityService) *AppointmentHandler {
	return &AppointmentHandler{svc: svc, availSvc: availSvc}
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

func (h *AppointmentHandler) HandleGetAvailability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	dateFromStr := r.URL.Query().Get("date_from")
	dateToStr := r.URL.Query().Get("date_to")
	
	// If date_from and date_to are missing, default to today if date is provided
	if dateFromStr == "" && dateToStr == "" {
		dateStr := r.URL.Query().Get("date")
		if dateStr != "" {
			dateFromStr = dateStr
			dateToStr = dateStr
		} else {
			myhttp.RespondError(w, http.StatusBadRequest, "date_from and date_to (or date) are required", "BAD_REQUEST", nil)
			return
		}
	}

	dateFrom, err := time.Parse("2006-01-02", dateFromStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid date_from format", "BAD_REQUEST", nil)
		return
	}

	dateTo, err := time.Parse("2006-01-02", dateToStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid date_to format", "BAD_REQUEST", nil)
		return
	}

	var docIDPtr *uuid.UUID
	if docIDStr := r.URL.Query().Get("doctor_id"); docIDStr != "" {
		id, err := uuid.Parse(docIDStr)
		if err == nil {
			docIDPtr = &id
		}
	}

	tz, _ := h.svc.repo.GetTenantTimezone(userCtx.TenantID)
	slots, err := h.availSvc.GetAvailableSlots(userCtx.TenantID, docIDPtr, dateFrom, dateTo, tz)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to get availability", "INTERNAL_ERROR", nil)
		return
	}

	// Wrapper to match frontend expected structure
	type responseWrapper struct {
		Data     []DoctorAvailabilityResponse `json:"data"`
		Timezone string                       `json:"timezone"`
		Message  string                       `json:"message"`
		Error    *string                      `json:"error"`
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseWrapper{
		Data:     slots,
		Timezone: tz,
		Message:  "success",
		Error:    nil,
	})
}

func (h *AppointmentHandler) HandleGetNextAvailable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	var docIDPtr *uuid.UUID
	if docIDStr := r.URL.Query().Get("doctor_id"); docIDStr != "" {
		id, err := uuid.Parse(docIDStr)
		if err == nil {
			docIDPtr = &id
		}
	}

	slot, err := h.availSvc.NextAvailableSlot(userCtx.TenantID, docIDPtr)
	if err != nil {
		myhttp.RespondError(w, http.StatusNotFound, "no slots available", "NOT_FOUND", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, slot, "success")
}
