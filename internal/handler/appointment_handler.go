package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"clinic-backend/internal/service"

	"github.com/google/uuid"
)

type AppointmentHandler struct {
	svc *service.AppointmentService
}

func NewAppointmentHandler(svc *service.AppointmentService) *AppointmentHandler {
	return &AppointmentHandler{svc: svc}
}

type ScheduleRequest struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	PatientID uuid.UUID `json:"patient_id"`
	DoctorID  uuid.UUID `json:"doctor_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

func (h *AppointmentHandler) HandleSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	appt, err := h.svc.ScheduleAppointment(req.TenantID, req.PatientID, req.DoctorID, req.StartTime, req.EndTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict) // Using 409 Conflict for double booking/invalid
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(appt)
}
