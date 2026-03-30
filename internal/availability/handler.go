package availability

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"

	"github.com/google/uuid"
)

// AvailabilityHandler routes HTTP requests to AvailabilityService.
// No business logic lives here; it only handles parsing, delegation, and response mapping.
//
// Route layout (all under /api/v1/doctors/{id}/availability):
//
//	GET    /api/v1/doctors/{id}/availability              → full schedule view
//	GET    /api/v1/doctors/{id}/availability/slots        → computed slot list
//	POST   /api/v1/doctors/{id}/availability/schedules    → create schedule entry
//	PATCH  /api/v1/doctors/{id}/availability/schedules/{sid} → update schedule
//	DELETE /api/v1/doctors/{id}/availability/schedules/{sid} → delete schedule
//	POST   /api/v1/doctors/{id}/availability/schedules/{sid}/breaks → add break
//	DELETE /api/v1/doctors/{id}/availability/breaks/{bid}   → delete break
//	GET    /api/v1/doctors/{id}/availability/exceptions     → list exceptions
//	POST   /api/v1/doctors/{id}/availability/exceptions     → create exception
//	DELETE /api/v1/doctors/{id}/availability/exceptions/{eid} → delete exception
type AvailabilityHandler struct {
	svc *AvailabilityService
}

// NewAvailabilityHandler constructs the handler with its required service.
func NewAvailabilityHandler(svc *AvailabilityService) *AvailabilityHandler {
	return &AvailabilityHandler{svc: svc}
}

// ─── Full Schedule View ───────────────────────────────────────────────────────

// HandleGetFullAvailability serves GET /api/v1/doctors/{id}/availability.
// Returns the complete availability configuration: schedules, breaks, and exceptions.
func (h *AvailabilityHandler) HandleGetFullAvailability(w http.ResponseWriter, r *http.Request) {
	userCtx, doctorID, ok := h.extractContext(w, r)
	if !ok {
		return
	}

	result, err := h.svc.GetFullAvailability(r.Context(), userCtx.TenantID, doctorID)
	if err != nil {
		h.respondDomainError(w, err)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, result, "success")
}

// ─── Available Slots ──────────────────────────────────────────────────────────

// HandleGetSlots serves GET /api/v1/doctors/{id}/availability/slots.
// Query params: date_from (YYYY-MM-DD), date_to (YYYY-MM-DD), slot_minutes (int, default 30).
func (h *AvailabilityHandler) HandleGetSlots(w http.ResponseWriter, r *http.Request) {
	userCtx, doctorID, ok := h.extractContext(w, r)
	if !ok {
		return
	}

	params, err := parseSlotQueryParams(r)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, err.Error(), "BAD_REQUEST", nil)
		return
	}
	params.DoctorID = doctorID

	result, err := h.svc.GetAvailableSlots(r.Context(), userCtx.TenantID, doctorID, params)
	if err != nil {
		h.respondDomainError(w, err)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, result, "success")
}

// ─── Schedules ────────────────────────────────────────────────────────────────

// HandleCreateSchedule serves POST /api/v1/doctors/{id}/availability/schedules.
func (h *AvailabilityHandler) HandleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	userCtx, doctorID, ok := h.extractContext(w, r)
	if !ok {
		return
	}

	var req CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", nil)
		return
	}

	dto, err := h.svc.CreateSchedule(r.Context(), userCtx.TenantID, doctorID, req)
	if err != nil {
		h.respondDomainError(w, err)
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, dto, "schedule created")
}

// HandleUpdateSchedule serves PATCH /api/v1/doctors/{id}/availability/schedules/{sid}.
func (h *AvailabilityHandler) HandleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	userCtx, _, ok := h.extractContext(w, r)
	if !ok {
		return
	}

	scheduleID, err := parsePathSegment(r.URL.Path, "schedules")
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid schedule id", "BAD_REQUEST", nil)
		return
	}

	var req UpdateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", nil)
		return
	}

	dto, err := h.svc.UpdateSchedule(r.Context(), userCtx.TenantID, scheduleID, req)
	if err != nil {
		h.respondDomainError(w, err)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, dto, "schedule updated")
}

// HandleDeleteSchedule serves DELETE /api/v1/doctors/{id}/availability/schedules/{sid}.
func (h *AvailabilityHandler) HandleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	userCtx, _, ok := h.extractContext(w, r)
	if !ok {
		return
	}

	scheduleID, err := parsePathSegment(r.URL.Path, "schedules")
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid schedule id", "BAD_REQUEST", nil)
		return
	}

	if err := h.svc.DeleteSchedule(r.Context(), userCtx.TenantID, scheduleID); err != nil {
		h.respondDomainError(w, err)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "schedule deleted")
}

// ─── Breaks ───────────────────────────────────────────────────────────────────

// HandleCreateBreak serves POST /api/v1/doctors/{id}/availability/schedules/{sid}/breaks.
func (h *AvailabilityHandler) HandleCreateBreak(w http.ResponseWriter, r *http.Request) {
	userCtx, doctorID, ok := h.extractContext(w, r)
	if !ok {
		return
	}

	scheduleID, err := parsePathSegment(r.URL.Path, "schedules")
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid schedule id", "BAD_REQUEST", nil)
		return
	}

	var req CreateBreakRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", nil)
		return
	}

	dto, err := h.svc.CreateBreak(r.Context(), userCtx.TenantID, doctorID, scheduleID, req)
	if err != nil {
		h.respondDomainError(w, err)
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, dto, "break created")
}

// HandleDeleteBreak serves DELETE /api/v1/doctors/{id}/availability/breaks/{bid}.
func (h *AvailabilityHandler) HandleDeleteBreak(w http.ResponseWriter, r *http.Request) {
	userCtx, _, ok := h.extractContext(w, r)
	if !ok {
		return
	}

	breakID, err := parsePathSegment(r.URL.Path, "breaks")
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid break id", "BAD_REQUEST", nil)
		return
	}

	if err := h.svc.DeleteBreak(r.Context(), userCtx.TenantID, breakID); err != nil {
		h.respondDomainError(w, err)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "break deleted")
}

// ─── Exceptions ───────────────────────────────────────────────────────────────

// HandleListExceptions serves GET /api/v1/doctors/{id}/availability/exceptions.
func (h *AvailabilityHandler) HandleListExceptions(w http.ResponseWriter, r *http.Request) {
	userCtx, doctorID, ok := h.extractContext(w, r)
	if !ok {
		return
	}

	dtos, err := h.svc.GetExceptionsByDoctor(r.Context(), userCtx.TenantID, doctorID)
	if err != nil {
		h.respondDomainError(w, err)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, dtos, "success")
}

// HandleCreateException serves POST /api/v1/doctors/{id}/availability/exceptions.
func (h *AvailabilityHandler) HandleCreateException(w http.ResponseWriter, r *http.Request) {
	userCtx, doctorID, ok := h.extractContext(w, r)
	if !ok {
		return
	}

	var req CreateExceptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", nil)
		return
	}

	dto, err := h.svc.CreateException(r.Context(), userCtx.TenantID, doctorID, req)
	if err != nil {
		h.respondDomainError(w, err)
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, dto, "exception created")
}

// HandleDeleteException serves DELETE /api/v1/doctors/{id}/availability/exceptions/{eid}.
func (h *AvailabilityHandler) HandleDeleteException(w http.ResponseWriter, r *http.Request) {
	userCtx, _, ok := h.extractContext(w, r)
	if !ok {
		return
	}

	exceptionID, err := parsePathSegment(r.URL.Path, "exceptions")
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid exception id", "BAD_REQUEST", nil)
		return
	}

	if err := h.svc.DeleteException(r.Context(), userCtx.TenantID, exceptionID); err != nil {
		h.respondDomainError(w, err)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "exception deleted")
}

// ─── Private helpers ──────────────────────────────────────────────────────────

// extractContext reads the auth context and extracts the doctor ID from the URL path.
// Returns false and writes the appropriate error response when either is missing.
func (h *AvailabilityHandler) extractContext(w http.ResponseWriter, r *http.Request) (*shared.UserContext, uuid.UUID, bool) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return nil, uuid.Nil, false
	}

	doctorID, err := parsePathSegment(r.URL.Path, "doctors")
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid doctor id", "BAD_REQUEST", nil)
		return nil, uuid.Nil, false
	}

	return userCtx, doctorID, true
}

// parsePathSegment extracts the UUID that immediately follows the given `segment`
// keyword in the URL path (e.g. "doctors" → /api/v1/doctors/{id}/…).
func parsePathSegment(urlPath, segment string) (uuid.UUID, error) {
	parts := strings.Split(urlPath, "/")
	for i, p := range parts {
		if p == segment && i+1 < len(parts) {
			return uuid.Parse(parts[i+1])
		}
	}
	return uuid.Nil, ErrNotFound
}

// parseSlotQueryParams validates and parses the slot query string parameters.
func parseSlotQueryParams(r *http.Request) (SlotQueryParams, error) {
	q := r.URL.Query()

	dateFromStr := q.Get("date_from")
	dateToStr := q.Get("date_to")

	if dateFromStr == "" || dateToStr == "" {
		return SlotQueryParams{}, errors.New("date_from and date_to are required")
	}

	dateFrom, err := time.Parse("2006-01-02", dateFromStr)
	if err != nil {
		return SlotQueryParams{}, errors.New("invalid date_from format, expected YYYY-MM-DD")
	}

	dateTo, err := time.Parse("2006-01-02", dateToStr)
	if err != nil {
		return SlotQueryParams{}, errors.New("invalid date_to format, expected YYYY-MM-DD")
	}

	slotDuration := 30 * time.Minute
	if minsStr := q.Get("slot_minutes"); minsStr != "" {
		var mins int
		if _, err := parseIntParam(minsStr, &mins); err == nil && mins > 0 {
			slotDuration = time.Duration(mins) * time.Minute
		}
	}

	return SlotQueryParams{
		DateFrom:     dateFrom,
		DateTo:       dateTo,
		SlotDuration: slotDuration,
	}, nil
}

// parseIntParam parses a string as an integer, writing the result into out.
func parseIntParam(s string, out *int) (string, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return s, err
	}
	*out = n
	return s, nil
}

// respondDomainError maps availability domain errors to HTTP status codes and
// structured error codes. Adding a new domain error only requires a case here.
func (h *AvailabilityHandler) respondDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		myhttp.RespondError(w, http.StatusNotFound, err.Error(), "NOT_FOUND", nil)
	case errors.Is(err, ErrOverlappingShift):
		myhttp.RespondError(w, http.StatusConflict, err.Error(), "OVERLAP_CONFLICT", nil)
	case errors.Is(err, ErrOverlappingBreak):
		myhttp.RespondError(w, http.StatusConflict, err.Error(), "BREAK_OUTSIDE_SHIFT", nil)
	case errors.Is(err, ErrInvalidTimeRange):
		myhttp.RespondError(w, http.StatusBadRequest, err.Error(), "INVALID_TIME_RANGE", nil)
	case errors.Is(err, ErrInvalidDayOfWeek):
		myhttp.RespondError(w, http.StatusBadRequest, err.Error(), "INVALID_DAY", nil)
	case errors.Is(err, ErrExceptionConflict):
		myhttp.RespondError(w, http.StatusConflict, err.Error(), "EXCEPTION_CONFLICT", nil)
	case errors.Is(err, ErrDoctorNotInTenant):
		myhttp.RespondError(w, http.StatusForbidden, err.Error(), "FORBIDDEN", nil)
	default:
		myhttp.RespondError(w, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR", nil)
	}
}
