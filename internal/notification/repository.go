package notification

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// NotificationRepository defines the data access contract for outbound notifications.
type NotificationRepository interface {
	CreateNotification(ctx context.Context, n *OutboundNotification) error
	UpdateDeliveryStatus(ctx context.Context, req UpdateDeliveryStatusRequest) error
	ListByPatient(ctx context.Context, tenantID, patientID uuid.UUID, limit, offset int) ([]*OutboundNotification, error)
	GetPreferences(ctx context.Context, tenantID, patientID uuid.UUID) (*PatientNotificationPreferences, error)
	UpsertPreferences(ctx context.Context, prefs *PatientNotificationPreferences) error
	IsReminderSent(ctx context.Context, appointmentID uuid.UUID, channel string) (bool, error)
	MarkReminderSent(ctx context.Context, tenantID, appointmentID uuid.UUID, channel string, scheduledFor time.Time) error
}

type postgresNotificationRepository struct {
	db *sql.DB
}

func NewPostgresNotificationRepository(db *sql.DB) NotificationRepository {
	return &postgresNotificationRepository{db: db}
}

func (r *postgresNotificationRepository) CreateNotification(ctx context.Context, n *OutboundNotification) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO outbound_notifications
		  (id, tenant_id, patient_id, appointment_id, channel, event_type,
		   recipient, subject, message, status, provider, scheduled_for, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW())
	`, n.ID, n.TenantID, n.PatientID, n.AppointmentID, n.Channel, n.EventType,
		n.Recipient, n.Subject, n.Message, n.Status, n.Provider, n.ScheduledFor)
	return err
}

func (r *postgresNotificationRepository) UpdateDeliveryStatus(ctx context.Context, req UpdateDeliveryStatusRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE outbound_notifications
		SET status = $1, provider_message_id = $2, error_message = $3, sent_at = $4
		WHERE id = $5
	`, req.Status, req.ProviderMessageID, req.ErrorMessage, req.SentAt, req.NotificationID)
	return err
}

func (r *postgresNotificationRepository) ListByPatient(ctx context.Context, tenantID, patientID uuid.UUID, limit, offset int) ([]*OutboundNotification, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, patient_id, appointment_id, channel, event_type,
		       recipient, subject, message, status, provider, provider_message_id,
		       error_message, scheduled_for, sent_at, created_at
		FROM outbound_notifications
		WHERE tenant_id = $1 AND patient_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, tenantID, patientID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*OutboundNotification
	for rows.Next() {
		var n OutboundNotification
		if err := rows.Scan(
			&n.ID, &n.TenantID, &n.PatientID, &n.AppointmentID,
			&n.Channel, &n.EventType, &n.Recipient, &n.Subject, &n.Message,
			&n.Status, &n.Provider, &n.ProviderMessageID,
			&n.ErrorMessage, &n.ScheduledFor, &n.SentAt, &n.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &n)
	}
	return list, rows.Err()
}

func (r *postgresNotificationRepository) GetPreferences(ctx context.Context, tenantID, patientID uuid.UUID) (*PatientNotificationPreferences, error) {
	var p PatientNotificationPreferences
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, patient_id,
		       email_enabled, whatsapp_enabled, reminder_enabled,
		       appointment_created_enabled, appointment_confirmed_enabled,
		       appointment_canceled_enabled, appointment_rescheduled_enabled,
		       created_at, updated_at
		FROM patient_notification_preferences
		WHERE tenant_id = $1 AND patient_id = $2
	`, tenantID, patientID).Scan(
		&p.ID, &p.TenantID, &p.PatientID,
		&p.EmailEnabled, &p.WhatsAppEnabled, &p.ReminderEnabled,
		&p.AppointmentCreatedEnabled, &p.AppointmentConfirmedEnabled,
		&p.AppointmentCanceledEnabled, &p.AppointmentRescheduledEnabled,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return defaultPreferences(tenantID, patientID), nil
	}
	return &p, err
}

func (r *postgresNotificationRepository) UpsertPreferences(ctx context.Context, p *PatientNotificationPreferences) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO patient_notification_preferences
		  (id, tenant_id, patient_id, email_enabled, whatsapp_enabled, reminder_enabled,
		   appointment_created_enabled, appointment_confirmed_enabled,
		   appointment_canceled_enabled, appointment_rescheduled_enabled,
		   created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),NOW())
		ON CONFLICT (tenant_id, patient_id) DO UPDATE SET
		  email_enabled = EXCLUDED.email_enabled,
		  whatsapp_enabled = EXCLUDED.whatsapp_enabled,
		  reminder_enabled = EXCLUDED.reminder_enabled,
		  appointment_created_enabled = EXCLUDED.appointment_created_enabled,
		  appointment_confirmed_enabled = EXCLUDED.appointment_confirmed_enabled,
		  appointment_canceled_enabled = EXCLUDED.appointment_canceled_enabled,
		  appointment_rescheduled_enabled = EXCLUDED.appointment_rescheduled_enabled,
		  updated_at = NOW()
	`, p.ID, p.TenantID, p.PatientID,
		p.EmailEnabled, p.WhatsAppEnabled, p.ReminderEnabled,
		p.AppointmentCreatedEnabled, p.AppointmentConfirmedEnabled,
		p.AppointmentCanceledEnabled, p.AppointmentRescheduledEnabled)
	return err
}

func (r *postgresNotificationRepository) IsReminderSent(ctx context.Context, appointmentID uuid.UUID, channel string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM appointment_reminders
		WHERE appointment_id = $1 AND channel = $2 AND sent_at IS NOT NULL
	`, appointmentID, channel).Scan(&count)
	return count > 0, err
}

func (r *postgresNotificationRepository) MarkReminderSent(ctx context.Context, tenantID, appointmentID uuid.UUID, channel string, scheduledFor time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO appointment_reminders (id, tenant_id, appointment_id, channel, scheduled_for, sent_at, created_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (appointment_id, channel) DO UPDATE SET sent_at = NOW()
	`, tenantID, appointmentID, channel, scheduledFor)
	return err
}

// defaultPreferences returns enabled-by-default prefs when no row exists.
func defaultPreferences(tenantID, patientID uuid.UUID) *PatientNotificationPreferences {
	return &PatientNotificationPreferences{
		ID:                           uuid.New(),
		TenantID:                     tenantID,
		PatientID:                    patientID,
		EmailEnabled:                 true,
		WhatsAppEnabled:              false,
		ReminderEnabled:              true,
		AppointmentCreatedEnabled:    true,
		AppointmentConfirmedEnabled:  true,
		AppointmentCanceledEnabled:   true,
		AppointmentRescheduledEnabled: true,
	}
}
