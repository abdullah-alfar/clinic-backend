package document

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Repository interface {
	Create(doc *Document) error
	GetByID(tenantID, id uuid.UUID) (*Document, error)
	ListByPatientID(tenantID, patientID uuid.UUID, category string) ([]Document, error)
	Update(doc *Document) error
	Delete(tenantID, id uuid.UUID) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(doc *Document) error {
	query := `
		INSERT INTO documents 
		(id, tenant_id, patient_id, appointment_id, medical_record_id, name, mime_type, size, storage_path, category, uploaded_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING created_at, updated_at`

	err := r.db.QueryRow(query,
		doc.ID, doc.TenantID, doc.PatientID, doc.AppointmentID, doc.MedicalRecordID,
		doc.Name, doc.MimeType, doc.Size, doc.StoragePath, doc.Category, doc.UploadedBy,
	).Scan(&doc.CreatedAt, &doc.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(tenantID, id uuid.UUID) (*Document, error) {
	query := `
		SELECT id, tenant_id, patient_id, appointment_id, medical_record_id, name, mime_type, size, storage_path, category, uploaded_by, created_at, updated_at
		FROM documents
		WHERE tenant_id = $1 AND id = $2`

	var doc Document
	err := r.db.QueryRow(query, tenantID, id).Scan(
		&doc.ID, &doc.TenantID, &doc.PatientID, &doc.AppointmentID, &doc.MedicalRecordID,
		&doc.Name, &doc.MimeType, &doc.Size, &doc.StoragePath, &doc.Category, &doc.UploadedBy,
		&doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &doc, nil
}

func (r *PostgresRepository) ListByPatientID(tenantID, patientID uuid.UUID, category string) ([]Document, error) {
	query := `
		SELECT id, tenant_id, patient_id, appointment_id, medical_record_id, name, mime_type, size, storage_path, category, uploaded_by, created_at, updated_at
		FROM documents 
		WHERE tenant_id = $1 AND patient_id = $2 `
	
	args := []interface{}{tenantID, patientID}
	
	if category != "" {
		query += "AND category = $3 "
		args = append(args, category)
	}
	
	query += "ORDER BY created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Document
	for rows.Next() {
		var doc Document
		if err := rows.Scan(
			&doc.ID, &doc.TenantID, &doc.PatientID, &doc.AppointmentID, &doc.MedicalRecordID,
			&doc.Name, &doc.MimeType, &doc.Size, &doc.StoragePath, &doc.Category, &doc.UploadedBy,
			&doc.CreatedAt, &doc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, doc)
	}
	return results, nil
}

func (r *PostgresRepository) Update(doc *Document) error {
	query := `
		UPDATE documents 
		SET name = $1, category = $2, appointment_id = $3, medical_record_id = $4, updated_at = NOW()
		WHERE tenant_id = $5 AND id = $6`

	_, err := r.db.Exec(query, 
		doc.Name, doc.Category, doc.AppointmentID, doc.MedicalRecordID,
		doc.TenantID, doc.ID,
	)
	return err
}

func (r *PostgresRepository) Delete(tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(`DELETE FROM documents WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	return err
}
