package document

import (
	"time"

	"github.com/google/uuid"
)

type DocumentResponse struct {
	ID              uuid.UUID        `json:"id"`
	PatientID       uuid.UUID        `json:"patient_id"`
	AppointmentID   *uuid.UUID       `json:"appointment_id,omitempty"`
	MedicalRecordID *uuid.UUID       `json:"medical_record_id,omitempty"`
	Name            string           `json:"name"`
	MimeType        string           `json:"mime_type"`
	Size            int64            `json:"size"`
	Category        DocumentCategory `json:"category"`
	UploadedBy      *uuid.UUID       `json:"uploaded_by,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	DownloadURL     string           `json:"download_url"`
}

type UpdateDocumentRequest struct {
	Name            string            `json:"name"`
	Category        DocumentCategory  `json:"category"`
	AppointmentID   *uuid.UUID       `json:"appointment_id,omitempty"`
	MedicalRecordID *uuid.UUID       `json:"medical_record_id,omitempty"`
}

func ToDocumentResponse(doc *Document) DocumentResponse {
	return DocumentResponse{
		ID:              doc.ID,
		PatientID:       doc.PatientID,
		AppointmentID:   doc.AppointmentID,
		MedicalRecordID: doc.MedicalRecordID,
		Name:            doc.Name,
		MimeType:        doc.MimeType,
		Size:            doc.Size,
		Category:        doc.Category,
		UploadedBy:      doc.UploadedBy,
		CreatedAt:       doc.CreatedAt,
		UpdatedAt:       doc.UpdatedAt,
		DownloadURL:     doc.StoragePath, // Or a specific API URL if preferred
	}
}
