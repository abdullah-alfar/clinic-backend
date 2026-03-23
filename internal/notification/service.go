package notification

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"-"`
	UserID            uuid.UUID  `json:"-"`
	Type              string     `json:"type"`
	Title             string     `json:"title"`
	Message           string     `json:"message"`
	Channel           string     `json:"channel"`
	Status            string     `json:"status"`
	RelatedEntityType *string    `json:"related_entity_type"`
	RelatedEntityID   *uuid.UUID `json:"related_entity_id"`
	ScheduledFor      *time.Time `json:"scheduled_for"`
	SentAt            *time.Time `json:"sent_at"`
	ReadAt            *time.Time `json:"read_at"`
	CreatedAt         time.Time  `json:"created_at"`
}

type NotificationService struct {
	db *sql.DB
}

func NewNotificationService(db *sql.DB) *NotificationService {
	return &NotificationService{db: db}
}

func (s *NotificationService) List(tenantID, userID uuid.UUID, limit, offset int) ([]*Notification, error) {
	rows, err := s.db.Query(`
		SELECT id, type, title, message, channel, status, 
		       related_entity_type, related_entity_id, scheduled_for, sent_at, read_at, created_at
		FROM notifications
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, tenantID, userID, limit, offset)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Notification
	for rows.Next() {
		var n Notification
		err := rows.Scan(
			&n.ID, &n.Type, &n.Title, &n.Message, &n.Channel, &n.Status,
			&n.RelatedEntityType, &n.RelatedEntityID, &n.ScheduledFor, &n.SentAt, &n.ReadAt, &n.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, &n)
	}
	return list, nil
}

func (s *NotificationService) MarkRead(tenantID, userID, notifID uuid.UUID) error {
	res, err := s.db.Exec(`
		UPDATE notifications SET read_at = NOW(), status = 'read'
		WHERE id = $1 AND tenant_id = $2 AND user_id = $3
	`, notifID, tenantID, userID)

	if err != nil {
		return err
	}

	count, _ := res.RowsAffected()
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}
