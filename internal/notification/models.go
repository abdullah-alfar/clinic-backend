package notification

import (
	"time"

	"github.com/google/uuid"
)

// OutboundNotification records a single outbound delivery attempt (email or whatsapp).
type OutboundNotification struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	PatientID         *uuid.UUID `json:"patient_id"`
	AppointmentID     *uuid.UUID `json:"appointment_id"`
	Channel           string     `json:"channel"`
	EventType         string     `json:"event_type"`
	Recipient         string     `json:"recipient"`
	Subject           *string    `json:"subject"`
	Message           string     `json:"message"`
	Status            string     `json:"status"`
	Provider          *string    `json:"provider"`
	ProviderMessageID *string    `json:"provider_message_id"`
	ErrorMessage      *string    `json:"error_message"`
	ScheduledFor      *time.Time `json:"scheduled_for"`
	SentAt            *time.Time `json:"sent_at"`
	CreatedAt         time.Time  `json:"created_at"`
}

// PatientNotificationPreferences holds per-patient opt-in/opt-out per channel and event.
type PatientNotificationPreferences struct {
	ID                           uuid.UUID `json:"id"`
	TenantID                     uuid.UUID `json:"tenant_id"`
	PatientID                    uuid.UUID `json:"patient_id"`
	EmailEnabled                 bool      `json:"email_enabled"`
	WhatsAppEnabled              bool      `json:"whatsapp_enabled"`
	ReminderEnabled              bool      `json:"reminder_enabled"`
	AppointmentCreatedEnabled    bool      `json:"appointment_created_enabled"`
	AppointmentConfirmedEnabled  bool      `json:"appointment_confirmed_enabled"`
	AppointmentCanceledEnabled   bool      `json:"appointment_canceled_enabled"`
	AppointmentRescheduledEnabled bool     `json:"appointment_rescheduled_enabled"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}
