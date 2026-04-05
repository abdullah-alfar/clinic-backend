package patient

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"net/http"
	"time"

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

func (h *PatientHandler) HandleUpdatePatient(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "failed to read request body", "BAD_REQUEST", nil)
		return
	}

	if len(body) == 0 {
		myhttp.RespondError(w, http.StatusBadRequest, "request body is empty", "BAD_REQUEST", nil)
		return
	}

	var req UpdatePatientRequest
	if err := json.Unmarshal(body, &req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid json body", "BAD_REQUEST", string(body))
		return
	}
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}
	id := r.PathValue("id")
	var dob *time.Time
	if req.DateOfBirth != nil && *req.DateOfBirth != "" {
		parsed, err := time.Parse("2006-01-02", *req.DateOfBirth)
		if err != nil {
			myhttp.RespondError(w, http.StatusBadRequest, "invalid date_of_birth format", "BAD_REQUEST", err.Error())
			return
		}
		dob = &parsed
	}

	p := Patient{
		ID:          uuid.MustParse(id),
		TenantID:    userCtx.TenantID,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Phone:       req.Phone,
		Email:       req.Email,
		DateOfBirth: dob,
		Gender:      req.Gender,
		Notes:       req.Notes,
	}

	if err := h.svc.UpdatePatient(&p, userCtx.UserID); err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to update patient", "UPDATE_FAILED", nil)
		return
	}
	myhttp.RespondJSON(w, http.StatusOK, p, "patient updated successfully")
}

func (h *PatientHandler) HandleDeletePatient(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	id := r.PathValue("id")

	patientID, err := uuid.Parse(id)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient id", "BAD_REQUEST", nil)
		return
	}

	if err := h.svc.DeletePatient(patientID, userCtx.TenantID); err != nil {
		switch {
		case errors.Is(err, ErrPatientNotFound):
			myhttp.RespondError(w, http.StatusNotFound, "patient not found", "NOT_FOUND", nil)
		default:
			myhttp.RespondError(w, http.StatusInternalServerError, "failed to delete patient", "DELETE_FAILED", nil)
		}
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "patient deleted successfully")
}
