package timeline

import (
	"net/http"
	"strconv"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
	"github.com/google/uuid"
)

type TimelineHandler struct {
	svc *TimelineService
}

func NewTimelineHandler(svc *TimelineService) *TimelineHandler {
	return &TimelineHandler{svc: svc}
}

func (h *TimelineHandler) HandlePatientTimeline(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	patientIDStr := r.PathValue("id")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient id", "BAD_REQUEST", err.Error())
		return
	}

	filterType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")
	limit := 0
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	timeline, err := h.svc.GetPatientTimeline(userCtx.TenantID, patientID, filterType, limit)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch timeline", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, timeline, "success")
}
