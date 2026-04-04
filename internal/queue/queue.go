package queue

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

// Task Types
const (
	TypeReminderEmail          = "email:reminder"
	TypeNotificationProcess    = "notification:process"
	TypeEmailNotification      = "notification:email"
	TypeWhatsAppNotification   = "notification:whatsapp"
	TypeAppointmentReminderMsg = "notification:appointment_reminder"
)

// Payloads
type ReminderEmailPayload struct {
	TenantID      string `json:"tenant_id"`
	AppointmentID string `json:"appointment_id"`
	PatientID     string `json:"patient_id"`
}

type NotificationPayload struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	Type     string `json:"type"`
}

type EmailNotificationPayload struct {
	NotificationID string `json:"notification_id"`
	TenantID       string `json:"tenant_id"`
	To             string `json:"to"`
	Subject        string `json:"subject"`
	HTMLBody       string `json:"html_body"`
	TextBody       string `json:"text_body"`
}

type WhatsAppNotificationPayload struct {
	NotificationID string `json:"notification_id"`
	TenantID       string `json:"tenant_id"`
	To             string `json:"to"`
	Body           string `json:"body"`
}

type QueueClient struct {
	client *asynq.Client
}

func NewQueueClient(redisAddr string) (*QueueClient, error) {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	return &QueueClient{client: client}, nil
}

func (q *QueueClient) EnqueueReminder(payload ReminderEmailPayload, sendAt time.Time) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeReminderEmail, bytes)
	_, err = q.client.Enqueue(task, asynq.ProcessAt(sendAt), asynq.MaxRetry(3))
	return err
}

func (q *QueueClient) EnqueueNotification(payload NotificationPayload) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeNotificationProcess, bytes)
	_, err = q.client.Enqueue(task, asynq.MaxRetry(3))
	return err
}

func (q *QueueClient) EnqueueEmailNotification(payload EmailNotificationPayload) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeEmailNotification, bytes)
	_, err = q.client.Enqueue(task, asynq.MaxRetry(3))
	return err
}

func (q *QueueClient) EnqueueWhatsAppNotification(payload WhatsAppNotificationPayload) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeWhatsAppNotification, bytes)
	_, err = q.client.Enqueue(task, asynq.MaxRetry(3))
	return err
}

func (q *QueueClient) Close() {
	if q.client != nil {
		q.client.Close()
	}
}
