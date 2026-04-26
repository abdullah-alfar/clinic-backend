package search

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type notificationProvider struct{ db *sql.DB }

// NewNotificationProvider creates a SearchProvider that searches system notifications.
func NewNotificationProvider(db *sql.DB) SearchProvider { return &notificationProvider{db: db} }

func (p *notificationProvider) Type() EntityType { return EntityNotification }
func (p *notificationProvider) Label() string    { return "Notifications" }

func (p *notificationProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResultItem, error) {
	pattern := "%" + req.Query + "%"

	args := []any{req.TenantID, pattern}
	extra := ""

	if req.Status != "" {
		args = append(args, req.Status)
		extra += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if req.DateFrom != nil {
		args = append(args, *req.DateFrom)
		extra += fmt.Sprintf(" AND created_at >= $%d", len(args))
	}
	if req.DateTo != nil {
		args = append(args, *req.DateTo)
		extra += fmt.Sprintf(" AND created_at <= $%d", len(args))
	}

	args = append(args, req.Limit)
	limitIdx := len(args)

	q := fmt.Sprintf(`
		SELECT id, type, title, message, status, created_at
		FROM notifications
		WHERE tenant_id = $1
		  AND (
		        type    ILIKE $2 OR
		        title   ILIKE $2 OR
		        message ILIKE $2 OR
		        status  ILIKE $2
		      )
		%s
		ORDER BY created_at DESC
		LIMIT $%d
	`, extra, limitIdx)

	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("notifications: %w", err)
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id, nType, title, message, status string
		var createdAt time.Time
		if err := rows.Scan(&id, &nType, &title, &message, &status, &createdAt); err != nil {
			return nil, fmt.Errorf("notifications scan: %w", err)
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       title,
			Subtitle:    message,
			Description: "Notification (" + nType + ")",
			URL:         "/notifications",
			Metadata: map[string]any{
				"type":       nType,
				"status":     status,
				"created_at": createdAt,
			},
		})
	}
	return results, rows.Err()
}
