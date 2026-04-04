package recurrence

import (
	"time"

	"github.com/google/uuid"
)

type Frequency string

const (
	FrequencyWeekly  Frequency = "weekly"
	FrequencyMonthly Frequency = "monthly"
)

type RecurrenceStatus string

const (
	StatusActive    RecurrenceStatus = "active"
	StatusCompleted RecurrenceStatus = "completed"
	StatusCancelled RecurrenceStatus = "cancelled"
)

type RecurrenceRule struct {
	ID          uuid.UUID        `json:"id"`
	TenantID    uuid.UUID        `json:"tenant_id"`
	PatientID   uuid.UUID        `json:"patient_id"`
	DoctorID    uuid.UUID        `json:"doctor_id"`
	Frequency   Frequency        `json:"frequency"`
	Interval    int              `json:"interval"`
	DayOfWeek   *int             `json:"day_of_week,omitempty"`  // (0-6)
	DayOfMonth  *int             `json:"day_of_month,omitempty"` // (1-31)
	StartTime   string           `json:"start_time"`            // "15:04:05"
	EndTime     string           `json:"end_time"`              // "15:04:05"
	StartDate   time.Time        `json:"start_date"`
	EndDate     time.Time        `json:"end_date"`
	Reason      string           `json:"reason"`
	Status      RecurrenceStatus `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type CreateRecurrenceRequest struct {
	PatientID       uuid.UUID `json:"patient_id"`
	DoctorID        uuid.UUID `json:"doctor_id"`
	Frequency       Frequency `json:"frequency"`
	Interval        int       `json:"interval"`
	DayOfWeek       *int      `json:"day_of_week,omitempty"`
	DayOfMonth      *int      `json:"day_of_month,omitempty"`
	StartTime       string    `json:"start_time"`
	EndTime         string    `json:"end_time"`
	StartDate       string    `json:"start_date"` // YYYY-MM-DD
	EndDate         string    `json:"end_date"`   // YYYY-MM-DD
	Reason          string    `json:"reason"`
}

type RecurrenceDTO struct {
	ID        uuid.UUID `json:"id"`
	Frequency Frequency `json:"frequency"`
	Summary   string    `json:"summary"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}
