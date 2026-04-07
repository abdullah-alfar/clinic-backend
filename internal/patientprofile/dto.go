package patientprofile

import (
	"clinic-backend/internal/patient"
	"time"

	"github.com/google/uuid"
)

type PatientProfileResponse struct {
	Data    PatientProfileData `json:"data"`
	Message string             `json:"message"`
	Error   *string            `json:"error"`
}

type PatientProfileData struct {
	Patient   PatientDTO    `json:"patient"`
	Flags     []PatientFlag   `json:"flags"`
}

type ActivityItemDTO struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"` // appointment, medical_record, invoice, communication
	Title     string    `json:"title"`
	Subtitle  string    `json:"subtitle"`
	Status    string    `json:"status"`
	OccurredAt time.Time `json:"occurred_at"`
}

type ActivityStreamResponse struct {
	Data       []ActivityItemDTO `json:"data"`
	TotalItems int               `json:"total_items"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	Message    string           `json:"message"`
}

type PatientDTO struct {
	ID          uuid.UUID  `json:"id"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	Phone       *string    `json:"phone"`
	Email       *string    `json:"email"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Gender      *string    `json:"gender"`
	CreatedAt   time.Time  `json:"created_at"`
}

type PatientRecentActivity struct {
	Appointments   []RecentActivity `json:"appointments"`
	MedicalRecords []RecentActivity `json:"medical_records"`
	Reports         []RecentActivity `json:"reports"`
	Invoices       []RecentActivity `json:"invoices"`
	Communications []RecentActivity `json:"communications"`
}

func FromPatientModel(p *patient.Patient) PatientDTO {
	return PatientDTO{
		ID:          p.ID,
		FirstName:   p.FirstName,
		LastName:    p.LastName,
		Phone:       p.Phone,
		Email:       p.Email,
		DateOfBirth: p.DateOfBirth,
		Gender:      p.Gender,
		CreatedAt:   p.CreatedAt,
	}
}
