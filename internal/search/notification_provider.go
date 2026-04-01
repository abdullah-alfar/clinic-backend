package search

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type notificationProvider struct {
	db *sql.DB
}

func NewNotificationProvider(db *sql.DB) SearchProvider {
	return &notificationProvider{db: db}
}

func (p *notificationProvider) GetEntityType() EntityType {
	return EntityNotification
}

func (p *notificationProvider) GetEntityLabel() string {
	return "Notifications"
}

func (p *notificationProvider) Search(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]SearchResultItem, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)

	q := `
		SELECT 
			id, 
			type, 
			title, 
			message, 
			status,
			created_at
		FROM notifications
		WHERE tenant_id = $1 
		  AND (
		      type ILIKE $2 OR 
		      title ILIKE $2 OR 
		      message ILIKE $2 OR
		      status ILIKE $2
		  )
		ORDER BY created_at DESC
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
		var nType, title, message, status string
		var createdAt interface{}
		if err := rows.Scan(&id, &nType, &title, &message, &status, &createdAt); err != nil {
			return nil, err
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       title,
			Subtitle:    message,
			Description: "Notification (" + nType + ")",
			URL:         "/notifications", // or specific notification link
			Score:       0,
			Metadata: map[string]any{
				"type":   nType,
				"status": status,
			},
		})
	}

	return results, nil
}
