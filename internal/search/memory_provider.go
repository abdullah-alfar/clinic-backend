package search

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type memoryProvider struct{ db *sql.DB }

// NewMemoryProvider creates a SearchProvider that searches AI report analyses (memory/insights).
func NewMemoryProvider(db *sql.DB) SearchProvider { return &memoryProvider{db: db} }

func (p *memoryProvider) Type() EntityType { return EntityMemory }
func (p *memoryProvider) Label() string    { return "AI Memory & Insights" }

func (p *memoryProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResultItem, error) {
	pattern := "%" + req.Query + "%"

	args := []any{req.TenantID, pattern}
	extra := ""

	if req.PatientID != nil {
		args = append(args, *req.PatientID)
		extra += fmt.Sprintf(" AND ra.patient_id = $%d", len(args))
	}
	if req.DateFrom != nil {
		args = append(args, *req.DateFrom)
		extra += fmt.Sprintf(" AND ra.created_at >= $%d", len(args))
	}
	if req.DateTo != nil {
		args = append(args, *req.DateTo)
		extra += fmt.Sprintf(" AND ra.created_at <= $%d", len(args))
	}

	args = append(args, req.Limit)
	limitIdx := len(args)

	q := fmt.Sprintf(`
		SELECT
			ra.id,
			ra.analysis_type,
			ra.summary,
			pt.first_name,
			pt.last_name,
			ra.attachment_id,
			ra.patient_id,
			ra.created_at
		FROM report_ai_analyses ra
		JOIN patients pt ON ra.patient_id = pt.id
		WHERE ra.tenant_id = $1
		  AND (
		        ra.analysis_type ILIKE $2 OR
		        ra.summary       ILIKE $2 OR
		        pt.first_name    ILIKE $2 OR
		        pt.last_name     ILIKE $2
		      )
		%s
		ORDER BY ra.created_at DESC
		LIMIT $%d
	`, extra, limitIdx)

	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("memory: %w", err)
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id, aType, summary, pFName, pLName, attachmentID, patientID string
		var createdAt time.Time
		if err := rows.Scan(&id, &aType, &summary, &pFName, &pLName, &attachmentID, &patientID, &createdAt); err != nil {
			return nil, fmt.Errorf("memory scan: %w", err)
		}

		subtitle := summary
		if len(subtitle) > 100 {
			subtitle = subtitle[:97] + "..."
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       "AI Insight: " + aType + " for " + pFName + " " + pLName,
			Subtitle:    subtitle,
			Description: "Medical Insight",
			URL:         fmt.Sprintf("/patients/%s?tab=reports", patientID),
			Metadata: map[string]any{
				"type":          aType,
				"attachment_id": attachmentID,
				"created_at":    createdAt,
			},
		})
	}
	return results, rows.Err()
}
