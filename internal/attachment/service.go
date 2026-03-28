package attachment

import (
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"clinic-backend/internal/audit"
)

type AttachmentService struct {
	repo    Repository
	storage StorageProvider
	audit   *audit.AuditService
}

func NewAttachmentService(repo Repository, storage StorageProvider, audit *audit.AuditService) *AttachmentService {
	return &AttachmentService{
		repo:    repo,
		storage: storage,
		audit:   audit,
	}
}

func (s *AttachmentService) UploadPatientFile(
	tenantID uuid.UUID,
	patientID uuid.UUID,
	appointmentID *uuid.UUID,
	uploaderID uuid.UUID,
	filename string,
	mimeType string,
	fileSize int64,
	reader io.Reader,
) (*Attachment, error) {
	
	fileID := uuid.New()
	
	// Save file to storage
	fileURL, err := s.storage.Save(tenantID, fileID, filename, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Simple fileType categorization based on mimeType
	fileType := "document"
	if strings.HasPrefix(mimeType, "image/") {
		fileType = "image"
	}

	att := &Attachment{
		ID:            fileID,
		TenantID:      tenantID,
		PatientID:     patientID,
		AppointmentID: appointmentID,
		Name:          filename,
		FileURL:       fileURL,
		FileType:      fileType,
		MimeType:      mimeType,
		FileSize:      fileSize,
	}

	if uploaderID != uuid.Nil {
		att.UploadedBy = &uploaderID
	}

	if err := s.repo.Create(att); err != nil {
		// Attempt rollback of file
		_ = s.storage.Delete(fileURL)
		return nil, fmt.Errorf("failed to save attachment metadata: %w", err)
	}

	if s.audit != nil {
		s.audit.LogAction(tenantID, uploaderID, "UPLOAD_ATTACHMENT", "patient", patientID, map[string]string{
			"attachment_id": fileID.String(),
			"file_name":     filename,
		})
	}

	return att, nil
}

func (s *AttachmentService) GetAttachment(tenantID, id uuid.UUID) (*Attachment, error) {
	att, err := s.repo.GetByID(tenantID, id)
	if err != nil {
		return nil, err
	}
	if att == nil {
		return nil, fmt.Errorf("attachment not found")
	}
	return att, nil
}

func (s *AttachmentService) GetPatientAttachments(tenantID, patientID uuid.UUID) ([]Attachment, error) {
	return s.repo.ListByPatientID(tenantID, patientID)
}

func (s *AttachmentService) DeleteAttachment(tenantID, id, userID uuid.UUID) error {
	att, err := s.GetAttachment(tenantID, id)
	if err != nil {
		return err
	}

	// Delete from storage
	if err := s.storage.Delete(att.FileURL); err != nil {
		// Log warning, but continue deleting metadata
		fmt.Printf("Warning: failed to delete file from storage: %v\n", err)
	}

	if err := s.repo.Delete(tenantID, id); err != nil {
		return err
	}

	if s.audit != nil {
		s.audit.LogAction(tenantID, userID, "DELETE_ATTACHMENT", "patient", att.PatientID, map[string]string{
			"attachment_id": id.String(),
			"file_name":     att.Name,
		})
	}

	return nil
}
