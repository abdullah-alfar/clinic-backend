package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"clinic-backend/internal/mail"
	"clinic-backend/internal/queue"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type Processor struct {
	db     *sql.DB
	mailer mail.Mailer
}

func NewProcessor(db *sql.DB, mailer mail.Mailer) *Processor {
	return &Processor{db: db, mailer: mailer}
}

func (p *Processor) HandleReminderEmail(ctx context.Context, t *asynq.Task) error {
	var payload queue.ReminderEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	// 1. Double check appointment status in DB (to support cancellation aborts)
	var status string
	var start time.Time
	var patientEmail *string
	var patientName string
	
	err := p.db.QueryRow(`
		SELECT a.status, a.start_time, p.email, p.first_name || ' ' || p.last_name
		FROM appointments a
		JOIN patients p ON a.patient_id = p.id
		WHERE a.id = $1 AND a.tenant_id = $2
	`, payload.AppointmentID, payload.TenantID).Scan(&status, &start, &patientEmail, &patientName)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[WORKER] Appointment %s not found. Skipping reminder.", payload.AppointmentID)
			return nil
		}
		return err // Retryable
	}

	// Don't remind for canceled/completed
	if status == "canceled" || status == "completed" {
		log.Printf("[WORKER] Appointment %s is %s. Skipping reminder.", payload.AppointmentID, status)
		return nil
	}

	// 2. Transmit Email gracefully via abstracted layer
	if patientEmail != nil && *patientEmail != "" {
		p.mailer.SendReminder(*patientEmail, patientName, start.Format(time.RFC822))
	} else {
		log.Printf("[WORKER] Patient %s has no email. Skipping transmission.", payload.PatientID)
	}

	return nil
}

func (p *Processor) HandleNotificationProcess(ctx context.Context, t *asynq.Task) error {
	var payload queue.NotificationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	// Write notification blindly to DB table
	_, err := p.db.Exec(`
		INSERT INTO notifications (id, tenant_id, user_id, type, title, message, channel, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`, uuid.New(), payload.TenantID, payload.UserID, payload.Type, payload.Title, payload.Message, "in_app", "pending")

	return err
}

func (p *Processor) Start(redisAddr string) error {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 10,
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Printf("[WORKER ERROR] Processing %s failed: %v", task.Type(), err)
				// Persist failed job logic here
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

	return srv.Run(mux)
}
