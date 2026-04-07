package ops_intelligence

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) HandleNoShowRisk(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := uuid.Parse(r.Header.Get("X-Tenant-ID")) // In real scenario, this comes from auth context
	
	// Assuming path /api/v1/appointments/{id}/no-show-risk
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	apptID, err := uuid.Parse(pathParts[4])
	if err != nil {
		http.Error(w, "invalid appointment id", http.StatusBadRequest)
		return
	}

	patientIDStr := r.URL.Query().Get("patient_id")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		http.Error(w, "invalid patient id", http.StatusBadRequest)
		return
	}

	risk, err := h.service.GetNoShowRisk(r.Context(), tenantID, apptID, patientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(risk)
}

func (h *Handler) HandleMissingRevenue(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	
	apptIDStr := r.URL.Query().Get("appointment_id")
	apptID, err := uuid.Parse(apptIDStr)
	if err != nil {
		http.Error(w, "invalid appointment id", http.StatusBadRequest)
		return
	}

	missing, err := h.service.GetMissingRevenue(r.Context(), tenantID, apptID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(missing)
}

func (h *Handler) HandleCommunications(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	
	patientIDStr := r.URL.Query().Get("patient_id")
	var patientID *uuid.UUID
	if pid, err := uuid.Parse(patientIDStr); err == nil {
		patientID = &pid
	}

	comms, err := h.service.GetCommunications(r.Context(), tenantID, patientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comms)
}
