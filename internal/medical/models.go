package medical

import (
	"time"

	"github.com/google/uuid"
)

type MedicalRecord struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	PatientID     uuid.UUID  `json:"patient_id"`
	AppointmentID *uuid.UUID `json:"appointment_id"`
	DoctorID      uuid.UUID  `json:"doctor_id"`
	Diagnosis     string     `json:"diagnosis"`
	Notes         string     `json:"notes"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type MedicalVital struct {
	ID              uuid.UUID `json:"id"`
	MedicalRecordID uuid.UUID `json:"medical_record_id"`
	Type            string    `json:"type"`
	Value           string    `json:"value"`
	Unit            *string   `json:"unit"`
	CreatedAt       time.Time `json:"created_at"`
}

type MedicalMedication struct {
	ID              uuid.UUID `json:"id"`
	MedicalRecordID uuid.UUID `json:"medical_record_id"`
	Name            string    `json:"name"`
	Dosage          string    `json:"dosage"`
	Frequency       string    `json:"frequency"`
	Duration        *string   `json:"duration"`
	Notes           *string   `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
}

type MedicalRecordProcedure struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	MedicalRecordID    uuid.UUID  `json:"medical_record_id"`
	ProcedureCatalogID uuid.UUID  `json:"procedure_catalog_id"`
	PerformedBy        *uuid.UUID `json:"performed_by,omitempty"`
	Notes              *string    `json:"notes,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}
