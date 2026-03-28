package invoice

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPatientNotFound = errors.New("patient not found or unauthorized")
	ErrApptNotFound    = errors.New("appointment not found or unauthorized")
	ErrInvoiceNotFound = errors.New("invoice not found")
	ErrInvalidStatus   = errors.New("invalid status transition")
)

type InvoiceService struct {
	repo InvoiceRepository
	db   *sql.DB // for validation lookups
}

func NewInvoiceService(repo InvoiceRepository, db *sql.DB) *InvoiceService {
	return &InvoiceService{repo: repo, db: db}
}

func (s *InvoiceService) CreateInvoice(req CreateInvoiceRequest, tenantID uuid.UUID) (*Invoice, error) {
	// Validate patient belongs to tenant
	var patientExists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM patients WHERE id = $1 AND tenant_id = $2)", req.PatientID, tenantID).Scan(&patientExists)
	if err != nil || !patientExists {
		return nil, ErrPatientNotFound
	}

	// Validate appointment if provided
	if req.AppointmentID != nil {
		var apptExists bool
		err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM appointments WHERE id = $1 AND patient_id = $2 AND tenant_id = $3)", *req.AppointmentID, req.PatientID, tenantID).Scan(&apptExists)
		if err != nil || !apptExists {
			return nil, ErrApptNotFound
		}
	}

	inv := &Invoice{
		ID:            uuid.New(),
		TenantID:      tenantID,
		PatientID:     req.PatientID,
		AppointmentID: req.AppointmentID,
		Amount:        req.Amount,
		Status:        "pending",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(inv); err != nil {
		return nil, err
	}

	return inv, nil
}

func (s *InvoiceService) ListPatientInvoices(patientID, tenantID uuid.UUID) ([]*Invoice, error) {
	// Validating patient existence is optional here, just returning empty array is fine
	return s.repo.ListByPatient(patientID, tenantID)
}

func (s *InvoiceService) MarkAsPaid(id, tenantID uuid.UUID) error {
	inv, err := s.repo.GetByIDAndTenant(id, tenantID)
	if err != nil {
		return err
	}
	if inv == nil {
		return ErrInvoiceNotFound
	}
	if inv.Status == "paid" {
		return nil // idempotent
	}

	return s.repo.UpdateStatus(id, tenantID, "paid")
}
