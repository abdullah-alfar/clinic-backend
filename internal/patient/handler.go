package patient

import (
	"encoding/json"
	"errors"
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
)

type PatientHandler struct {
	svc *PatientService
}

func NewPatientHandler(svc *PatientService) *PatientHandler {
	return &PatientHandler{svc: svc}
}

func (h *PatientHandler) HandlePatients(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		patients, err := h.svc.ListPatients(userCtx.TenantID)
		if err != nil {
			myhttp.RespondError(w, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR", nil)
			return
		}
		myhttp.RespondJSON(w, http.StatusOK, patients, "success")

	case http.MethodPost:
		var p Patient
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", err.Error())
			return
		}

		p.TenantID = userCtx.TenantID // Enforce tenant isolation

		if err := h.svc.CreatePatient(&p, userCtx.UserID); err != nil {
			myhttp.RespondError(w, http.StatusInternalServerError, "failed to create patient", "CREATION_FAILED", nil)
			return
		}
		myhttp.RespondJSON(w, http.StatusCreated, p, "patient created successfully")

	default:
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
	}
}

func (h *PatientHandler) HandlePatientByID(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	id := r.PathValue("id")

	patient, err := h.svc.GetPatientByID(id, userCtx.TenantID)
	if err != nil {
		switch {
		case errors.Is(err, ErrPatientNotFound):
			myhttp.RespondError(w, http.StatusNotFound, "patient not found", "NOT_FOUND", nil)
		default:
			myhttp.RespondError(w, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR", nil)
		}
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, patient, "success")
}
