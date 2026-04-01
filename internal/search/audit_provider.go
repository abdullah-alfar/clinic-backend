package search

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type auditProvider struct {
	db *sql.DB
}

func NewAuditProvider(db *sql.DB) SearchProvider {
	return &auditProvider{db: db}
}

func (p *auditProvider) GetEntityType() EntityType {
	return EntityAudit
}

func (p *auditProvider) GetEntityLabel() string {
	return "Audit Activity"
}

func (p *auditProvider) Search(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]SearchResultItem, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)

	q := `
		SELECT 
			a.id, 
			a.action, 
			a.entity_type, 
			a.entity_id, 
			u.name,
			a.created_at,
			a.metadata
		FROM audit_logs a
		JOIN users u ON a.user_id = u.id
		WHERE a.tenant_id = $1 
		  AND (
		      a.action ILIKE $2 OR 
		      a.entity_type ILIKE $2 OR 
		      u.name ILIKE $2 OR 
		      CAST(a.metadata AS TEXT) ILIKE $2
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
		var id, action, entityType, entityID, uName string
		var createdAt interface{}
		var metadata sql.NullString
		if err := rows.Scan(&id, &action, &entityType, &entityID, &uName, &createdAt, &metadata); err != nil {
			return nil, err
		}

		title := fmt.Sprintf("%s was %s", entityType, action)
		subtitle := "By " + uName

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       title,
			Subtitle:    subtitle,
			Description: "Audit Log Entry",
			URL:         fmt.Sprintf("/%s/%s", entityType, entityID), // hypothetical link
			Score:       0,
			Metadata: map[string]any{
				"entity_id":   entityID,
				"action":      action,
				"entity_type": entityType,
			},
		})
	}

	return results, nil
}
