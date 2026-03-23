package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"clinic-backend/internal/models"

	"github.com/google/uuid"
)

type PatientHandler struct {
	patients map[uuid.UUID]*models.Patient
}

func NewPatientHandler() *PatientHandler {
	return &PatientHandler{
		patients: make(map[uuid.UUID]*models.Patient),
	}
}

func (h *PatientHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var p models.Patient
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	h.patients[p.ID] = &p

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func (h *PatientHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	list := make([]*models.Patient, 0, len(h.patients))
	for _, p := range h.patients {
		list = append(list, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
