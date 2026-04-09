package document

import (
	"fmt"
	"io"

	"github.com/google/uuid"
	"clinic-backend/internal/attachment" // Reuse StorageProvider
	"clinic-backend/internal/audit"
)

type DocumentService struct {
	repo    Repository
	storage attachment.StorageProvider
	audit   *audit.AuditService
}

func NewDocumentService(repo Repository, storage attachment.StorageProvider, audit *audit.AuditService) *DocumentService {
	return &DocumentService{
		repo:    repo,
		storage: storage,
		audit:   audit,
	}
}

func (s *DocumentService) UploadDocument(
	tenantID uuid.UUID,
	patientID uuid.UUID,
	appointmentID *uuid.UUID,
	medicalRecordID *uuid.UUID,
	uploaderID uuid.UUID,
	name string,
	category DocumentCategory,
	mimeType string,
	size int64,
	reader io.Reader,
) (*Document, error) {
	
	docID := uuid.New()
	
	// Save file to storage
	storagePath, err := s.storage.Save(tenantID, docID, name, reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageFailed, err)
	}

	doc := &Document{
		ID:              docID,
		TenantID:        tenantID,
		PatientID:       patientID,
		AppointmentID:   appointmentID,
		MedicalRecordID: medicalRecordID,
		Name:            name,
		MimeType:        mimeType,
		Size:            size,
		StoragePath:     storagePath,
		Category:        category,
	}

	if uploaderID != uuid.Nil {
		doc.UploadedBy = &uploaderID
	}

	if err := s.repo.Create(doc); err != nil {
		_ = s.storage.Delete(storagePath)
		return nil, fmt.Errorf("failed to save document metadata: %w", err)
	}

	if s.audit != nil {
		s.audit.LogAction(tenantID, uploaderID, "UPLOAD_DOCUMENT", "patient", patientID, map[string]string{
			"document_id": docID.String(),
			"name":        name,
		})
	}

	return doc, nil
}

func (s *DocumentService) GetPatientDocuments(tenantID, patientID uuid.UUID, category string) ([]Document, error) {
	return s.repo.ListByPatientID(tenantID, patientID, category)
}

func (s *DocumentService) UpdateDocument(tenantID, id, userID uuid.UUID, req UpdateDocumentRequest) (*Document, error) {
	doc, err := s.repo.GetByID(tenantID, id)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, ErrDocumentNotFound
	}

	doc.Name = req.Name
	doc.Category = req.Category
	doc.AppointmentID = req.AppointmentID
	doc.MedicalRecordID = req.MedicalRecordID

	if err := s.repo.Update(doc); err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.LogAction(tenantID, userID, "UPDATE_DOCUMENT", "patient", doc.PatientID, map[string]string{
			"document_id": id.String(),
		})
	}

	return doc, nil
}

func (s *DocumentService) DeleteDocument(tenantID, id, userID uuid.UUID) error {
	doc, err := s.repo.GetByID(tenantID, id)
	if err != nil {
		return err
	}
	if doc == nil {
		return ErrDocumentNotFound
	}

	if err := s.storage.Delete(doc.StoragePath); err != nil {
		// Log warning but continue
		fmt.Printf("Warning: failed to delete storage file %s: %v\n", doc.StoragePath, err)
	}

	if err := s.repo.Delete(tenantID, id); err != nil {
		return err
	}

	if s.audit != nil {
		s.audit.LogAction(tenantID, userID, "DELETE_DOCUMENT", "patient", doc.PatientID, map[string]string{
			"document_id": id.String(),
			"name":        doc.Name,
		})
	}

	return nil
}

func (s *DocumentService) GetDocumentByID(tenantID, id uuid.UUID) (*Document, error) {
	doc, err := s.repo.GetByID(tenantID, id)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, ErrDocumentNotFound
	}
	return doc, nil
}
