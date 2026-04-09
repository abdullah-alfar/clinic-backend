package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type visitNoteProvider struct {
	db *sql.DB
}

func NewVisitNoteProvider(db *sql.DB) SearchProvider {
	return &visitNoteProvider{db: db}
}

func (p *visitNoteProvider) GetEntityType() EntityType {
	return EntityNote
}

func (p *visitNoteProvider) GetEntityLabel() string {
	return "Medical Notes"
}

func (p *visitNoteProvider) Search(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]SearchResultItem, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)

	q := `
		SELECT 
			v.id, 
			v.notes, 
			v.diagnosis, 
			v.prescription, 
			pt.first_name, 
			pt.last_name,
			v.created_at
		FROM visits v
		JOIN patients pt ON v.patient_id = pt.id
		WHERE v.tenant_id = $1 
		  AND (
		      v.notes ILIKE $2 OR 
		      v.diagnosis ILIKE $2 OR 
		      v.prescription ILIKE $2 OR
		      pt.first_name ILIKE $2 OR 
		      pt.last_name ILIKE $2 OR
		      (pt.first_name || ' ' || pt.last_name) ILIKE $2
		  )
		ORDER BY v.created_at DESC
		LIMIT $3
	`

	rows, err := p.db.QueryContext(ctx, q, tenantID, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id string
		var notes, diagnosis, prescription sql.NullString
		var pFName, pLName string
		var createdAt interface{}
		if err := rows.Scan(&id, &notes, &diagnosis, &prescription, &pFName, &pLName, &createdAt); err != nil {
			return nil, err
		}

		// Pick the most relevant title/subtitle
		title := "Visit for " + pFName + " " + pLName
		var contents []string
		if diagnosis.Valid && diagnosis.String != "" {
			contents = append(contents, "Dx: "+diagnosis.String)
		}
		if prescription.Valid && prescription.String != "" {
			contents = append(contents, "Rx: "+prescription.String)
		}
		
		subtitle := strings.Join(contents, " • ")
		if subtitle == "" && notes.Valid {
			subtitle = notes.String
			if len(subtitle) > 50 {
				subtitle = subtitle[:47] + "..."
			}
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       title,
			Subtitle:    subtitle,
			Description: "Visit Record",
			URL:         fmt.Sprintf("/patients/%s?tab=timeline", id), // or logic to view specific visit
			Score:       0,
			Metadata: map[string]any{
				"has_prescription": prescription.Valid && prescription.String != "",
			},
		})
	}

	return results, nil
}
