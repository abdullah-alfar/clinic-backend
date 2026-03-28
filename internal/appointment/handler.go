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

// AppointmentHandler routes HTTP requests to the AppointmentService.
// No business logic lives here; it only handles parsing, delegation, and response mapping.
type AppointmentHandler struct {
	svc      *AppointmentService
	availSvc *AvailabilityService
}

func NewAppointmentHandler(svc *AppointmentService, availSvc *AvailabilityService) *AppointmentHandler {
	return &AppointmentHandler{svc: svc, availSvc: availSvc}
}

// ScheduleRequest is the body for POST /appointments.
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
		h.respondSchedulingError(w, err)
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, appt, "appointment scheduled successfully")
}

// UpdateTimeRequest is the body for PATCH /appointments/{id}.
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

	apptID, err := parsePathID(r.URL.Path, 4)
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

// HandleReschedule serves PATCH /appointments/{id}/reschedule.
// This is the dedicated endpoint for drag-and-drop rescheduling.
// It returns typed error codes so the frontend can show meaningful messages.
func (h *AppointmentHandler) HandleReschedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	apptID, err := parsePathID(r.URL.Path, 4)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid appointment id", "BAD_REQUEST", nil)
		return
	}

	var req RescheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", err.Error())
		return
	}

	if err := h.svc.RescheduleAppointment(userCtx.TenantID, apptID, req.StartTime, req.EndTime, userCtx.UserID); err != nil {
		h.respondRescheduleError(w, err)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "appointment rescheduled successfully")
}

// HandleGetCalendar serves GET /appointments/calendar.
// Returns enriched appointments (with patient and doctor names) for a date range.
func (h *AppointmentHandler) HandleGetCalendar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	params, err := parseCalendarQueryParams(r)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, err.Error(), "BAD_REQUEST", nil)
		return
	}

	appointments, timezone, err := h.svc.GetCalendarAppointments(userCtx.TenantID, params)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch calendar appointments", "INTERNAL_ERROR", nil)
		return
	}

	dtos := make([]CalendarAppointmentDTO, 0, len(appointments))
	for _, a := range appointments {
		dtos = append(dtos, toCalendarDTO(a))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CalendarResponse{
		Data:     dtos,
		Timezone: timezone,
		Message:  "success",
		Error:    nil,
	})
}

// HandleStatus returns a handler that transitions an appointment to the given status.
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

		apptID, err := parsePathID(r.URL.Path, 4)
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

// --- Private helpers ---

// parsePathID extracts a UUID from position `index` of a slash-split URL path.
func parsePathID(urlPath string, index int) (uuid.UUID, error) {
	parts := strings.Split(urlPath, "/")
	if len(parts) <= index {
		return uuid.Nil, ErrNotFound
	}
	return uuid.Parse(parts[index])
}

// parseCalendarQueryParams reads and validates query parameters for the calendar endpoint.
func parseCalendarQueryParams(r *http.Request) (CalendarQueryParams, error) {
	q := r.URL.Query()

	dateFromStr := q.Get("date_from")
	dateToStr := q.Get("date_to")

	if dateFromStr == "" || dateToStr == "" {
		return CalendarQueryParams{}, ErrInvalidTime
	}

	dateFrom, err := time.Parse("2006-01-02", dateFromStr)
	if err != nil {
		return CalendarQueryParams{}, ErrInvalidTime
	}

	dateTo, err := time.Parse("2006-01-02", dateToStr)
	if err != nil {
		return CalendarQueryParams{}, ErrInvalidTime
	}

	params := CalendarQueryParams{DateFrom: dateFrom, DateTo: dateTo}

	if docIDStr := q.Get("doctor_id"); docIDStr != "" {
		id, err := uuid.Parse(docIDStr)
		if err == nil {
			params.DoctorID = &id
		}
	}

	return params, nil
}

// respondSchedulingError maps domain errors from scheduling operations to HTTP responses.
func (h *AppointmentHandler) respondSchedulingError(w http.ResponseWriter, err error) {
	switch err {
	case ErrDoubleBooking:
		myhttp.RespondError(w, http.StatusConflict, err.Error(), "DOUBLE_BOOKING", nil)
	case ErrDoctorInactive:
		myhttp.RespondError(w, http.StatusConflict, err.Error(), "OUTSIDE_AVAILABILITY", nil)
	case ErrInvalidTime:
		myhttp.RespondError(w, http.StatusBadRequest, err.Error(), "INVALID_TIME", nil)
	case ErrPastAppointment:
		myhttp.RespondError(w, http.StatusUnprocessableEntity, err.Error(), "PAST_APPOINTMENT", nil)
	default:
		myhttp.RespondError(w, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR", nil)
	}
}

// respondRescheduleError maps domain errors from rescheduling to typed HTTP responses.
// Typed error codes allow the frontend to show contextual messages per failure reason.
func (h *AppointmentHandler) respondRescheduleError(w http.ResponseWriter, err error) {
	switch err {
	case ErrDoubleBooking:
		myhttp.RespondError(w, http.StatusConflict, err.Error(), "DOUBLE_BOOKING", nil)
	case ErrDoctorInactive:
		myhttp.RespondError(w, http.StatusConflict, err.Error(), "OUTSIDE_AVAILABILITY", nil)
	case ErrNotMutable:
		myhttp.RespondError(w, http.StatusUnprocessableEntity, err.Error(), "NOT_MUTABLE", nil)
	case ErrNotFound:
		myhttp.RespondError(w, http.StatusNotFound, err.Error(), "NOT_FOUND", nil)
	case ErrInvalidTime:
		myhttp.RespondError(w, http.StatusBadRequest, err.Error(), "INVALID_TIME", nil)
	case ErrPastAppointment:
		myhttp.RespondError(w, http.StatusUnprocessableEntity, err.Error(), "PAST_APPOINTMENT", nil)
	default:
		myhttp.RespondError(w, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR", nil)
	}
}
