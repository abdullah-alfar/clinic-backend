package scheduling

import (
	"time"

	"github.com/google/uuid"
)

// Strategy represents the ranking strategy for suggestions.
type Strategy string

const (
	StrategyFastest Strategy = "fastest" // Earliest available time
	StrategyBestFit Strategy = "best_fit" // Minimizes gaps in the schedule
)

// SlotSuggestion represents a recommended appointment window.
type SlotSuggestion struct {
	DoctorID   uuid.UUID `json:"doctor_id"`
	DoctorName string    `json:"doctor_name"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Score      float64   `json:"score"`
	Reason     string    `json:"reason"`
}

// SuggestionRequest contains filters for smart scheduling.
type SuggestionRequest struct {
	PatientID       uuid.UUID `json:"patient_id"`
	DoctorID        *uuid.UUID `json:"doctor_id,omitempty"`
	Specialty       string    `json:"specialty,omitempty"`
	DateFrom        time.Time `json:"date_from"`
	DateTo          time.Time `json:"date_to"`
	DurationMinutes int       `json:"duration_minutes"`
	Strategy        Strategy  `json:"strategy"`
}

type SuggestionResponse struct {
	Data    []SlotSuggestion `json:"data"`
	Message string           `json:"message"`
	Error   *string          `json:"error"`
}
