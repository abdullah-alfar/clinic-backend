package medical

import (
	"encoding/json"
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
	"github.com/google/uuid"
)

type MedicalHandler struct {
	svc *MedicalService
}

func NewMedicalHandler(svc *MedicalService) *MedicalHandler {
	return &MedicalHandler{svc: svc}
}

func (h *MedicalHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/patients/{id}/medical-records", h.ListRecords)
	mux.HandleFunc("POST /api/v1/patients/{id}/medical-records", h.CreateRecord)
	mux.HandleFunc("GET /api/v1/medical-records/{id}", h.GetRecord)
	mux.HandleFunc("PATCH /api/v1/medical-records/{id}", h.UpdateRecord)
	mux.HandleFunc("DELETE /api/v1/medical-records/{id}", h.DeleteRecord)
}

func (h *MedicalHandler) ListRecords(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	patientIDRaw := r.PathValue("id")
	patientID, err := uuid.Parse(patientIDRaw)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient id", "BAD_REQUEST", err.Error())
		return
	}

	records, err := h.svc.ListRecordsByPatient(userCtx.TenantID, patientID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to get records", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, records, "success")
}

func (h *MedicalHandler) CreateRecord(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	patientIDRaw := r.PathValue("id")
	patientID, err := uuid.Parse(patientIDRaw)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient id", "BAD_REQUEST", err.Error())
		return
	}

	var req CreateMedicalRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", err.Error())
		return
	}

	resp, err := h.svc.CreateRecord(userCtx.TenantID, userCtx.UserID, patientID, req)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to create record", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, resp, "medical record created")
}

func (h *MedicalHandler) GetRecord(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	recordIDRaw := r.PathValue("id")
	recordID, err := uuid.Parse(recordIDRaw)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid record id", "BAD_REQUEST", err.Error())
		return
	}

	resp, err := h.svc.GetRecord(userCtx.TenantID, recordID)
	if err != nil {
		if err == ErrRecordNotFound {
			myhttp.RespondError(w, http.StatusNotFound, "record not found", "NOT_FOUND", nil)
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to get record", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, resp, "success")
}

func (h *MedicalHandler) UpdateRecord(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	recordIDRaw := r.PathValue("id")
	recordID, err := uuid.Parse(recordIDRaw)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid record id", "BAD_REQUEST", err.Error())
		return
	}

	var req UpdateMedicalRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", err.Error())
		return
	}

	resp, err := h.svc.UpdateRecord(userCtx.TenantID, userCtx.UserID, recordID, req)
	if err != nil {
		if err == ErrRecordNotFound {
			myhttp.RespondError(w, http.StatusNotFound, "record not found", "NOT_FOUND", nil)
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to update record", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, resp, "medical record updated")
}

func (h *MedicalHandler) DeleteRecord(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	recordIDRaw := r.PathValue("id")
	recordID, err := uuid.Parse(recordIDRaw)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid record id", "BAD_REQUEST", err.Error())
		return
	}

	err = h.svc.DeleteRecord(userCtx.TenantID, userCtx.UserID, recordID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to delete record", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "medical record deleted")
}
