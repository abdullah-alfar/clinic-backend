package notification

// Appointment lifecycle event constants — used across dispatcher, template builder, and preferences.
const (
	EventAppointmentCreated     = "appointment_created"
	EventAppointmentConfirmed   = "appointment_confirmed"
	EventAppointmentCanceled    = "appointment_canceled"
	EventAppointmentRescheduled = "appointment_rescheduled"
	EventAppointmentReminder    = "appointment_reminder"
)

// Channel constants for outbound delivery.
const (
	ChannelEmail    = "email"
	ChannelWhatsApp = "whatsapp"
	ChannelInApp    = "in_app"
)

// Delivery status constants.
const (
	StatusPending = "pending"
	StatusSent    = "sent"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
)
