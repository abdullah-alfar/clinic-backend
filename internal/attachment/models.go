package attachment

import (
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	PatientID     uuid.UUID  `json:"patient_id"`
	AppointmentID *uuid.UUID `json:"appointment_id,omitempty"`
	Name          string     `json:"name"`
	FileURL       string     `json:"file_url"`
	FileType      string     `json:"file_type"`
	MimeType      string     `json:"mime_type"`
	FileSize      int64      `json:"file_size"`
	UploadedBy    *uuid.UUID `json:"uploaded_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
