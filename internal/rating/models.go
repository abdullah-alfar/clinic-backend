package rating

import (
	"time"

	"github.com/google/uuid"
)

type Rating struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	PatientID     uuid.UUID `json:"patient_id"`
	DoctorID      uuid.UUID `json:"doctor_id"`
	AppointmentID uuid.UUID `json:"appointment_id"`
	Rating        int       `json:"rating"`
	Comment       string    `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
}

type DoctorRatingSummary struct {
	AverageRating float64 `json:"average_rating"`
	TotalRatings  int     `json:"total_ratings"`
	Distribution  map[int]int `json:"distribution"` // e.g. {5: 10, 4: 2, ...}
}
