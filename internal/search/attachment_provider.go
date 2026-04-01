package search

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type attachmentProvider struct {
	db *sql.DB
}

func NewAttachmentProvider(db *sql.DB) SearchProvider {
	return &attachmentProvider{db: db}
}

func (p *attachmentProvider) GetEntityType() EntityType {
	return EntityReport
}

func (p *attachmentProvider) GetEntityLabel() string {
	return "Reports & Attachments"
}

func (p *attachmentProvider) Search(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]SearchResultItem, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)

	q := `
		SELECT 
			a.id, 
			a.name, 
			a.mime_type, 
			pt.first_name, 
			pt.last_name
		FROM attachments a
		JOIN patients pt ON a.patient_id = pt.id
		WHERE a.tenant_id = $1 
		  AND (
		      a.name ILIKE $2 OR 
		      a.mime_type ILIKE $2 OR 
		      pt.first_name ILIKE $2 OR 
		      pt.last_name ILIKE $2
		  )
		ORDER BY a.created_at DESC
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
		var name, mimeType, pFName, pLName string
		if err := rows.Scan(&id, &name, &mimeType, &pFName, &pLName); err != nil {
			return nil, err
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       name,
			Subtitle:    fmt.Sprintf("%s • Patient: %s %s", mimeType, pFName, pLName),
			Description: "Report",
			URL:         fmt.Sprintf("/patients/%s?tab=reports", id), // or logic to view the attachment directly
			Score:       0,
			Metadata: map[string]any{
				"mime_type": mimeType,
			},
		})
	}

	return results, nil
}
