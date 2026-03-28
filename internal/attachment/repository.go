package attachment

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Repository interface {
	Create(att *Attachment) error
	GetByID(tenantID, id uuid.UUID) (*Attachment, error)
	ListByPatientID(tenantID, patientID uuid.UUID) ([]Attachment, error)
	Delete(tenantID, id uuid.UUID) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(att *Attachment) error {
	query := `
		INSERT INTO attachments 
		(id, tenant_id, patient_id, appointment_id, name, file_url, file_type, mime_type, file_size, uploaded_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING created_at, updated_at`

	err := r.db.QueryRow(query,
		att.ID, att.TenantID, att.PatientID, att.AppointmentID,
		att.Name, att.FileURL, att.FileType, att.MimeType, att.FileSize, att.UploadedBy,
	).Scan(&att.CreatedAt, &att.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert attachment: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(tenantID, id uuid.UUID) (*Attachment, error) {
	query := `
		SELECT id, tenant_id, patient_id, appointment_id, name, file_url, file_type, mime_type, file_size, uploaded_by, created_at, updated_at
		FROM attachments
		WHERE tenant_id = $1 AND id = $2`

	var att Attachment
	err := r.db.QueryRow(query, tenantID, id).Scan(
		&att.ID, &att.TenantID, &att.PatientID, &att.AppointmentID,
		&att.Name, &att.FileURL, &att.FileType, &att.MimeType, &att.FileSize, &att.UploadedBy,
		&att.CreatedAt, &att.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &att, nil
}

func (r *PostgresRepository) ListByPatientID(tenantID, patientID uuid.UUID) ([]Attachment, error) {
	query := `
		SELECT id, tenant_id, patient_id, appointment_id, name, file_url, file_type, mime_type, file_size, uploaded_by, created_at, updated_at
		FROM attachments 
		WHERE tenant_id = $1 AND patient_id = $2
		ORDER BY created_at DESC`

	rows, err := r.db.Query(query, tenantID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Attachment
	for rows.Next() {
		var att Attachment
		if err := rows.Scan(
			&att.ID, &att.TenantID, &att.PatientID, &att.AppointmentID,
			&att.Name, &att.FileURL, &att.FileType, &att.MimeType, &att.FileSize, &att.UploadedBy,
			&att.CreatedAt, &att.UpdatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, att)
	}
	return results, nil
}

func (r *PostgresRepository) Delete(tenantID, id uuid.UUID) error {
	res, err := r.db.Exec(`DELETE FROM attachments WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("attachment not found or not belonging to tenant")
	}
	return nil
}
