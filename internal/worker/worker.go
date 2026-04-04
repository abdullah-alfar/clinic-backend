package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"clinic-backend/internal/mail"
	"clinic-backend/internal/notification"
	"clinic-backend/internal/queue"
	"clinic-backend/internal/whatsapp"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type Processor struct {
	db       *sql.DB
	mailer   mail.Mailer // Legacy
	email    mail.EmailSender
	WhatsApp whatsapp.WhatsAppSender
	notifRepo notification.NotificationRepository
}

func NewProcessor(db *sql.DB, mailer mail.Mailer, email mail.EmailSender, wa whatsapp.WhatsAppSender, notifRepo notification.NotificationRepository) *Processor {
	return &Processor{
		db:       db,
		mailer:   mailer,
		email:    email,
		WhatsApp: wa,
		notifRepo: notifRepo,
	}
}

func (p *Processor) HandleReminderEmail(ctx context.Context, t *asynq.Task) error {
	var payload queue.ReminderEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	var status string
	var start time.Time
	var patientEmail *string
	var patientName string
	
	err := p.db.QueryRow(`
		SELECT a.status, a.start_time, pat.email, pat.first_name || ' ' || pat.last_name
		FROM appointments a
		JOIN patients pat ON a.patient_id = pat.id
		WHERE a.id = $1 AND a.tenant_id = $2
	`, payload.AppointmentID, payload.TenantID).Scan(&status, &start, &patientEmail, &patientName)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[WORKER] Appointment %s not found. Skipping reminder.", payload.AppointmentID)
			return nil
		}
		return err
	}

	if status == "canceled" || status == "completed" {
		return nil
	}

	if patientEmail != nil && *patientEmail != "" {
		p.mailer.SendReminder(*patientEmail, patientName, start.Format(time.RFC822))
	}
	return nil
}

func (p *Processor) HandleNotificationProcess(ctx context.Context, t *asynq.Task) error {
	var payload queue.NotificationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	_, err := p.db.Exec(`
		INSERT INTO notifications (id, tenant_id, user_id, type, title, message, channel, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`, uuid.New(), payload.TenantID, payload.UserID, payload.Type, payload.Title, payload.Message, "in_app", "pending")

	return err
}

func (p *Processor) HandleEmailNotification(ctx context.Context, t *asynq.Task) error {
	var payload queue.EmailNotificationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	notifID, _ := uuid.Parse(payload.NotificationID)
	
	msg := mail.EmailMessage{
		To:       payload.To,
		Subject:  payload.Subject,
		TextBody: payload.TextBody,
		HTMLBody: payload.HTMLBody,
	}

	err := p.email.Send(ctx, msg)

	now := time.Now()
	status := notification.StatusSent
	var errMsg *string
	if err != nil {
		status = notification.StatusFailed
		e := err.Error()
		errMsg = &e
	}

	p.notifRepo.UpdateDeliveryStatus(context.Background(), notification.UpdateDeliveryStatusRequest{
		NotificationID: notifID,
		Status:         status,
		ErrorMessage:   errMsg,
		SentAt:         &now,
	})

	return err
}

func (p *Processor) HandleWhatsAppNotification(ctx context.Context, t *asynq.Task) error {
	var payload queue.WhatsAppNotificationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	notifID, _ := uuid.Parse(payload.NotificationID)

	// Normalize phone number before sending
	phone, err := whatsapp.NormalizePhone(payload.To)
	if err != nil {
		e := err.Error()
		p.notifRepo.UpdateDeliveryStatus(context.Background(), notification.UpdateDeliveryStatusRequest{
			NotificationID: notifID,
			Status:         notification.StatusFailed,
			ErrorMessage:   &e,
		})
		return nil // Don't retry invalid phone numbers
	}
	
	msg := whatsapp.WhatsAppMessage{
		To:   phone,
		Body: payload.Body,
	}

	providerMsgID, err := p.WhatsApp.Send(ctx, msg)

	now := time.Now()
	status := notification.StatusSent
	var errMsg *string
	var pMsgID *string

	if err != nil {
		status = notification.StatusFailed
		e := err.Error()
		errMsg = &e
	} else {
		pMsgID = &providerMsgID
	}

	p.notifRepo.UpdateDeliveryStatus(context.Background(), notification.UpdateDeliveryStatusRequest{
		NotificationID:    notifID,
		Status:            status,
		ProviderMessageID: pMsgID,
		ErrorMessage:      errMsg,
		SentAt:            &now,
	})

	return err
}

func (p *Processor) Start(redisAddr string) error {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 10,
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Printf("[WORKER ERROR] Processing %s failed: %v", task.Type(), err)
				p.db.Exec(`
					INSERT INTO failed_jobs (id, job_type, payload, error_message, failed_at)
					VALUES ($1, $2, $3, $4, NOW())
				`, uuid.New(), task.Type(), task.Payload(), err.Error())
			}),
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TypeReminderEmail, p.HandleReminderEmail)
	mux.HandleFunc(queue.TypeNotificationProcess, p.HandleNotificationProcess)
	mux.HandleFunc(queue.TypeEmailNotification, p.HandleEmailNotification)
	mux.HandleFunc(queue.TypeWhatsAppNotification, p.HandleWhatsAppNotification)

	return srv.Run(mux)
}
