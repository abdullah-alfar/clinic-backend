package search

import (
	"context"
	"database/sql"
	"fmt"
)

type attachmentProvider struct{ db *sql.DB }

// NewAttachmentProvider creates a SearchProvider that searches patient attachments/reports.
func NewAttachmentProvider(db *sql.DB) SearchProvider { return &attachmentProvider{db: db} }

func (p *attachmentProvider) Type() EntityType { return EntityReport }
func (p *attachmentProvider) Label() string    { return "Reports & Attachments" }

func (p *attachmentProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResultItem, error) {
	pattern := "%" + req.Query + "%"

	args := []any{req.TenantID, pattern}
	extra := ""

	if req.PatientID != nil {
		args = append(args, *req.PatientID)
		extra += fmt.Sprintf(" AND a.patient_id = $%d", len(args))
	}
	if req.DateFrom != nil {
		args = append(args, *req.DateFrom)
		extra += fmt.Sprintf(" AND a.created_at >= $%d", len(args))
	}
	if req.DateTo != nil {
		args = append(args, *req.DateTo)
		extra += fmt.Sprintf(" AND a.created_at <= $%d", len(args))
	}

	args = append(args, req.Limit)
	limitIdx := len(args)

	q := fmt.Sprintf(`
		SELECT
			a.id,
			a.name,
			a.mime_type,
			pt.first_name,
			pt.last_name,
			a.patient_id
		FROM attachments a
		JOIN patients pt ON a.patient_id = pt.id
		WHERE a.tenant_id = $1
		  AND (
		        a.name      ILIKE $2 OR
		        a.mime_type ILIKE $2 OR
		        pt.first_name ILIKE $2 OR
		        pt.last_name  ILIKE $2
		      )
		%s
		ORDER BY a.created_at DESC
		LIMIT $%d
	`, extra, limitIdx)

	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("attachments: %w", err)
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id, name, mimeType, pFName, pLName, patientID string
		if err := rows.Scan(&id, &name, &mimeType, &pFName, &pLName, &patientID); err != nil {
			return nil, fmt.Errorf("attachments scan: %w", err)
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       name,
			Subtitle:    mimeType + " • Patient: " + pFName + " " + pLName,
			Description: "Report",
			URL:         fmt.Sprintf("/patients/%s?tab=reports", patientID),
			Metadata:    map[string]any{"mime_type": mimeType},
		})
	}
	return results, rows.Err()
}
