package document

import (
	"time"

	"github.com/google/uuid"
)

type DocumentCategory string

const (
	CategoryLabReport    DocumentCategory = "lab_report"
	CategoryPrescription DocumentCategory = "prescription"
	CategoryIDDocument   DocumentCategory = "id_document"
	CategoryInsurance    DocumentCategory = "insurance"
	CategoryConsentForm  DocumentCategory = "consent_form"
	CategoryGeneral      DocumentCategory = "general"
)

type Document struct {
	ID              uuid.UUID        `json:"id"`
	TenantID        uuid.UUID        `json:"tenant_id"`
	PatientID       uuid.UUID        `json:"patient_id"`
	AppointmentID   *uuid.UUID       `json:"appointment_id,omitempty"`
	MedicalRecordID *uuid.UUID       `json:"medical_record_id,omitempty"`
	Name            string           `json:"name"`
	MimeType        string           `json:"mime_type"`
	Size            int64            `json:"size"`
	StoragePath     string           `json:"storage_path"`
	Category        DocumentCategory `json:"category"`
	UploadedBy      *uuid.UUID       `json:"uploaded_by,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}
