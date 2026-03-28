package invoice

import (
	"database/sql"

	"github.com/google/uuid"
)

type InvoiceRepository interface {
	Create(invoice *Invoice) error
	GetByIDAndTenant(id, tenantID uuid.UUID) (*Invoice, error)
	ListByPatient(patientID, tenantID uuid.UUID) ([]*Invoice, error)
	UpdateStatus(id, tenantID uuid.UUID, status string) error
}

type postgresInvoiceRepository struct {
	db *sql.DB
}

func NewPostgresInvoiceRepository(db *sql.DB) InvoiceRepository {
	return &postgresInvoiceRepository{db: db}
}

func (r *postgresInvoiceRepository) Create(inv *Invoice) error {
	_, err := r.db.Exec(`
		INSERT INTO invoices (id, tenant_id, patient_id, appointment_id, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, inv.ID, inv.TenantID, inv.PatientID, inv.AppointmentID, inv.Amount, inv.Status, inv.CreatedAt, inv.UpdatedAt)
	return err
}

func (r *postgresInvoiceRepository) GetByIDAndTenant(id, tenantID uuid.UUID) (*Invoice, error) {
	var inv Invoice
	err := r.db.QueryRow(`
		SELECT id, tenant_id, patient_id, appointment_id, amount, status, created_at, updated_at
		FROM invoices
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(
		&inv.ID, &inv.TenantID, &inv.PatientID, &inv.AppointmentID,
		&inv.Amount, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &inv, nil
}

func (r *postgresInvoiceRepository) ListByPatient(patientID, tenantID uuid.UUID) ([]*Invoice, error) {
	rows, err := r.db.Query(`
		SELECT id, tenant_id, patient_id, appointment_id, amount, status, created_at, updated_at
		FROM invoices
		WHERE patient_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC
	`, patientID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*Invoice
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.PatientID, &inv.AppointmentID,
			&inv.Amount, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		invoices = append(invoices, &inv)
	}
	return invoices, nil
}

func (r *postgresInvoiceRepository) UpdateStatus(id, tenantID uuid.UUID, status string) error {
	_, err := r.db.Exec(`
		UPDATE invoices
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`, status, id, tenantID)
	return err
}
