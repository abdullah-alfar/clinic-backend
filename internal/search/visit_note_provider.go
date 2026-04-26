package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type visitNoteProvider struct{ db *sql.DB }

// NewVisitNoteProvider creates a SearchProvider that searches visit records and clinical notes.
func NewVisitNoteProvider(db *sql.DB) SearchProvider { return &visitNoteProvider{db: db} }

func (p *visitNoteProvider) Type() EntityType { return EntityNote }
func (p *visitNoteProvider) Label() string    { return "Medical Notes" }

func (p *visitNoteProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResultItem, error) {
	pattern := "%" + req.Query + "%"

	args := []any{req.TenantID, pattern}
	extra := ""

	if req.PatientID != nil {
		args = append(args, *req.PatientID)
		extra += fmt.Sprintf(" AND v.patient_id = $%d", len(args))
	}
	if req.DoctorID != nil {
		args = append(args, *req.DoctorID)
		extra += fmt.Sprintf(" AND v.doctor_id = $%d", len(args))
	}
	if req.DateFrom != nil {
		args = append(args, *req.DateFrom)
		extra += fmt.Sprintf(" AND v.created_at >= $%d", len(args))
	}
	if req.DateTo != nil {
		args = append(args, *req.DateTo)
		extra += fmt.Sprintf(" AND v.created_at <= $%d", len(args))
	}

	args = append(args, req.Limit)
	limitIdx := len(args)

	q := fmt.Sprintf(`
		SELECT
			v.id,
			v.notes,
			v.diagnosis,
			v.prescription,
			pt.first_name,
			pt.last_name,
			v.patient_id,
			v.created_at
		FROM visits v
		JOIN patients pt ON v.patient_id = pt.id
		WHERE v.tenant_id = $1
		  AND (
		        v.notes        ILIKE $2 OR
		        v.diagnosis    ILIKE $2 OR
		        v.prescription ILIKE $2 OR
		        pt.first_name  ILIKE $2 OR
		        pt.last_name   ILIKE $2 OR
		        (pt.first_name || ' ' || pt.last_name) ILIKE $2
		      )
		%s
		ORDER BY v.created_at DESC
		LIMIT $%d
	`, extra, limitIdx)

	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("visit notes: %w", err)
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id, pFName, pLName, patientID string
		var notes, diagnosis, prescription sql.NullString
		var createdAt time.Time

		if err := rows.Scan(&id, &notes, &diagnosis, &prescription, &pFName, &pLName, &patientID, &createdAt); err != nil {
			return nil, fmt.Errorf("visit notes scan: %w", err)
		}

		var parts []string
		if diagnosis.Valid && diagnosis.String != "" {
			parts = append(parts, "Dx: "+diagnosis.String)
		}
		if prescription.Valid && prescription.String != "" {
			parts = append(parts, "Rx: "+prescription.String)
		}
		subtitle := strings.Join(parts, " • ")
		if subtitle == "" && notes.Valid && notes.String != "" {
			subtitle = notes.String
			if len(subtitle) > 80 {
				subtitle = subtitle[:77] + "..."
			}
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       "Visit for " + pFName + " " + pLName,
			Subtitle:    subtitle,
			Description: "Visit Record",
			URL:         fmt.Sprintf("/patients/%s?tab=timeline", patientID),
			Metadata: map[string]any{
				"has_prescription": prescription.Valid && prescription.String != "",
				"created_at":       createdAt,
			},
		})
	}
	return results, rows.Err()
}
