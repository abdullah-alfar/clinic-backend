package reportai

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type Repository interface {
	Create(analysis *ReportAIAnalysis) error
	GetByAttachmentID(tenantID, attachmentID uuid.UUID) ([]ReportAIAnalysis, error)
	UpdateStatus(analysis *ReportAIAnalysis) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(a *ReportAIAnalysis) error {
	query := `
		INSERT INTO report_ai_analyses 
		(id, tenant_id, patient_id, attachment_id, analysis_type, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING created_at, updated_at`

	err := r.db.QueryRow(query,
		a.ID, a.TenantID, a.PatientID, a.AttachmentID, a.AnalysisType, a.Status, a.CreatedBy,
	).Scan(&a.CreatedAt, &a.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create report ai analysis: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByAttachmentID(tenantID, attachmentID uuid.UUID) ([]ReportAIAnalysis, error) {
	query := `
		SELECT id, tenant_id, patient_id, attachment_id, analysis_type, status, summary, structured_data, error_message, created_by, created_at, updated_at
		FROM report_ai_analyses
		WHERE tenant_id = $1 AND attachment_id = $2
		ORDER BY created_at DESC`

	rows, err := r.db.Query(query, tenantID, attachmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ReportAIAnalysis
	for rows.Next() {
		var a ReportAIAnalysis
		var summary, errorMsg sql.NullString
		var structData []byte

		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.PatientID, &a.AttachmentID, &a.AnalysisType, &a.Status,
			&summary, &structData, &errorMsg, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if summary.Valid {
			a.Summary = &summary.String
		}
		if errorMsg.Valid {
			a.ErrorMessage = &errorMsg.String
		}
		if len(structData) > 0 {
			a.StructuredData = json.RawMessage(structData)
		}

		results = append(results, a)
	}
	return results, nil
}

func (r *PostgresRepository) UpdateStatus(a *ReportAIAnalysis) error {
	query := `
		UPDATE report_ai_analyses
		SET status = $2,
		    summary = $3,
		    structured_data = $4,
		    raw_response = $5,
		    error_message = $6,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	var structData, rawData interface{}
	if a.StructuredData != nil {
		structData = a.StructuredData
	}
	if a.RawResponse != nil {
		rawData = a.RawResponse
	}

	err := r.db.QueryRow(query,
		a.ID, a.Status, a.Summary, structData, rawData, a.ErrorMessage,
	).Scan(&a.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to update report analysis: %w", err)
	}
	return nil
}
