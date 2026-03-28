package invoice

import (
	"time"

	"github.com/google/uuid"
)

type Invoice struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"-"`
	PatientID     uuid.UUID  `json:"patient_id"`
	AppointmentID *uuid.UUID `json:"appointment_id"`
	Amount        float64    `json:"amount"`
	Status        string     `json:"status"` // "pending", "paid"
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
