package search

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type memoryProvider struct {
	db *sql.DB
}

func NewMemoryProvider(db *sql.DB) SearchProvider {
	return &memoryProvider{db: db}
}

func (p *memoryProvider) GetEntityType() EntityType {
	return EntityMemory
}

func (p *memoryProvider) GetEntityLabel() string {
	return "AI Memory & Insights"
}

func (p *memoryProvider) Search(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]SearchResultItem, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)

	q := `
		SELECT 
			ra.id, 
			ra.analysis_type, 
			ra.summary, 
			pt.first_name, 
			pt.last_name,
			ra.attachment_id
		FROM report_ai_analyses ra
		JOIN patients pt ON ra.patient_id = pt.id
		WHERE ra.tenant_id = $1 
		  AND (
		      ra.analysis_type ILIKE $2 OR 
		      ra.summary ILIKE $2 OR 
		      pt.first_name ILIKE $2 OR 
		      pt.last_name ILIKE $2
		  )
		ORDER BY ra.created_at DESC
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
		var aType, summary, pFName, pLName, attachmentID string
		if err := rows.Scan(&id, &aType, &summary, &pFName, &pLName, &attachmentID); err != nil {
			return nil, err
		}

		title := "AI Insight: " + aType + " for " + pFName + " " + pLName
		subtitle := summary
		if len(subtitle) > 100 {
			subtitle = subtitle[:97] + "..."
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       title,
			Subtitle:    subtitle,
			Description: "Medical Insight",
			URL:         fmt.Sprintf("/patients/%s?tab=reports", id), // or link to specific analysis
			Score:       0,
			Metadata: map[string]any{
				"type":          aType,
				"attachment_id": attachmentID,
			},
		})
	}

	return results, nil
}
