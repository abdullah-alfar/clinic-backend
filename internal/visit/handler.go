package visit

import (
	"encoding/json"
	"net/http"

	"clinic-backend/internal/models"
	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
)

type VisitHandler struct {
	svc *VisitService
}

func NewVisitHandler(svc *VisitService) *VisitHandler {
	return &VisitHandler{svc: svc}
}

func (h *VisitHandler) HandleVisits(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	if r.Method != http.MethodPost {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED", nil)
		return
	}

	var v models.Visit
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", err.Error())
		return
	}

	v.TenantID = userCtx.TenantID
	v.DoctorID = userCtx.UserID

	if err := h.svc.CreateVisit(&v, userCtx.UserID); err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to create visit", "CREATION_FAILED", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, v, "visit created successfully")
}

func (h *VisitHandler) HandlePatientTimeline(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	patientID := r.PathValue("id")
	if patientID == "" {
		myhttp.RespondError(w, http.StatusBadRequest, "missing patient id", "BAD_REQUEST", nil)
		return
	}

	timeline, err := h.svc.GetPatientTimeline(patientID, userCtx.TenantID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch timeline", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, timeline, "success")
}
