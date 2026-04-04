package notification

import (
	"time"

	"github.com/google/uuid"
)

// AppointmentEventPayload carries everything needed to dispatch notifications
// for an appointment lifecycle event. Resolved once by the appointment service.
type AppointmentEventPayload struct {
	TenantID      uuid.UUID
	PatientID     uuid.UUID
	AppointmentID uuid.UUID
	ActorID       uuid.UUID
	Event         string
	// Enriched data for template building
	PatientName  string
	PatientEmail string
	PatientPhone string
	DoctorName   string
	ClinicName   string
	StartTime    time.Time
	EndTime      time.Time
	Timezone     string
}

// UpdateDeliveryStatusRequest is used by workers to report delivery outcomes.
type UpdateDeliveryStatusRequest struct {
	NotificationID    uuid.UUID
	Status            string
	ProviderMessageID *string
	ErrorMessage      *string
	SentAt            *time.Time
}

// UpsertPreferencesRequest is the DTO for the preferences PUT endpoint.
type UpsertPreferencesRequest struct {
	EmailEnabled                 bool `json:"email_enabled"`
	WhatsAppEnabled              bool `json:"whatsapp_enabled"`
	ReminderEnabled              bool `json:"reminder_enabled"`
	AppointmentCreatedEnabled    bool `json:"appointment_created_enabled"`
	AppointmentConfirmedEnabled  bool `json:"appointment_confirmed_enabled"`
	AppointmentCanceledEnabled   bool `json:"appointment_canceled_enabled"`
	AppointmentRescheduledEnabled bool `json:"appointment_rescheduled_enabled"`
}
