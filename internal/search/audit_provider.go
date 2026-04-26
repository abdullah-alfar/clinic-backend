package search

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type auditProvider struct{ db *sql.DB }

// NewAuditProvider creates a SearchProvider that searches audit log entries.
func NewAuditProvider(db *sql.DB) SearchProvider { return &auditProvider{db: db} }

func (p *auditProvider) Type() EntityType { return EntityAudit }
func (p *auditProvider) Label() string    { return "Audit Activity" }

func (p *auditProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResultItem, error) {
	pattern := "%" + req.Query + "%"

	args := []any{req.TenantID, pattern}
	extra := ""

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
		        a.action      ILIKE $2 OR
		        a.entity_type ILIKE $2 OR
		        u.name        ILIKE $2 OR
		        CAST(a.metadata AS TEXT) ILIKE $2
		      )
		%s
		ORDER BY a.created_at DESC
		LIMIT $%d
	`, extra, limitIdx)

	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("audit: %w", err)
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id, action, entityType, entityID, uName string
		var createdAt time.Time
		var metadata sql.NullString
		if err := rows.Scan(&id, &action, &entityType, &entityID, &uName, &createdAt, &metadata); err != nil {
			return nil, fmt.Errorf("audit scan: %w", err)
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       entityType + " was " + action,
			Subtitle:    "By " + uName,
			Description: "Audit Log Entry",
			URL:         fmt.Sprintf("/%s/%s", entityType, entityID),
			Metadata: map[string]any{
				"entity_id":   entityID,
				"action":      action,
				"entity_type": entityType,
				"created_at":  createdAt,
			},
		})
	}
	return results, rows.Err()
}
