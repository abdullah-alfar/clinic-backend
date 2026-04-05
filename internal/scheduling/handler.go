package scheduling

import (
	"fmt"
	"net/http"
	"time"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"

	"github.com/google/uuid"
)

type SmartSchedulingHandler struct {
	service *SmartSchedulingService
}

func NewSmartSchedulingHandler(service *SmartSchedulingService) *SmartSchedulingHandler {
	return &SmartSchedulingHandler{service: service}
}

func (h *SmartSchedulingHandler) HandleSmartSuggestions(w http.ResponseWriter, r *http.Request) {
	uctx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "missing user context", "UNAUTHORIZED", nil)
		return
	}

	query := r.URL.Query()

	// Parse parameters
	patientIDStr := query.Get("patient_id")
	patientID, _ := uuid.Parse(patientIDStr)

	doctorIDStr := query.Get("doctor_id")
	var doctorID *uuid.UUID
	if doctorIDStr != "" && doctorIDStr != "undefined" {
		if dID, err := uuid.Parse(doctorIDStr); err == nil {
			doctorID = &dID
		}
	}

	dateFromStr := query.Get("date_from")
	dateFrom, _ := time.Parse("2006-01-02", dateFromStr)
	if dateFrom.IsZero() {
		dateFrom = time.Now()
	}

	dateToStr := query.Get("date_to")
	dateTo, _ := time.Parse("2006-01-02", dateToStr)
	if dateTo.IsZero() {
		dateTo = dateFrom.AddDate(0, 0, 7) // Default 7 days
	}

	durationStr := query.Get("duration_minutes")
	var duration int
	if durationStr != "" {
		fmt.Sscanf(durationStr, "%d", &duration)
	}
	if duration <= 0 {
		duration = 30
	}

	strategy := Strategy(query.Get("strategy"))
	if strategy == "" {
		strategy = StrategyFastest
	}

	req := SuggestionRequest{
		PatientID:       patientID,
		DoctorID:        doctorID,
		DateFrom:        dateFrom,
		DateTo:          dateTo,
		DurationMinutes: duration,
		Strategy:        strategy,
	}

	suggestions, err := h.service.SuggestSlots(r.Context(), uctx.TenantID, req)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to suggest slots", "INTERNAL_ERROR", err)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, suggestions, "success")
}
