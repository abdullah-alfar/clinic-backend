package models

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Subdomain   string         `json:"subdomain"`
	Timezone    string         `json:"timezone"`
	ThemeConfig map[string]any `json:"theme_config"`
	CreatedAt   time.Time      `json:"created_at"`
}

type User struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Patient struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	ContactInfo string    `json:"contact_info"`
	CreatedAt   time.Time `json:"created_at"`
}

type AppointmentStatus string

const (
	StatusScheduled AppointmentStatus = "scheduled"
	StatusConfirmed AppointmentStatus = "confirmed"
	StatusCanceled  AppointmentStatus = "canceled"
	StatusCompleted AppointmentStatus = "completed"
)

type Appointment struct {
	ID        uuid.UUID         `json:"id"`
	TenantID  uuid.UUID         `json:"tenant_id"`
	PatientID uuid.UUID         `json:"patient_id"`
	DoctorID  uuid.UUID         `json:"doctor_id"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
	Status    AppointmentStatus `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
}

type Visit struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	PatientID     uuid.UUID `json:"patient_id"`
	AppointmentID uuid.UUID `json:"appointment_id"`
	DoctorID      uuid.UUID `json:"doctor_id"`
	Notes         string    `json:"notes"`
	Diagnosis     string    `json:"diagnosis"`
	Prescription  string    `json:"prescription"`
	CreatedAt     time.Time `json:"created_at"`
}
