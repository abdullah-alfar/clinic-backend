package whatsapp

// Template name constants — must match approved templates in the provider dashboard.
const (
	TemplateLang                  = "en"
	TemplateAppointmentCreated    = "appointment_created"
	TemplateAppointmentConfirmed  = "appointment_confirmed"
	TemplateAppointmentCanceled   = "appointment_canceled"
	TemplateAppointmentRescheduled = "appointment_rescheduled"
	TemplateAppointmentReminder   = "appointment_reminder"
)

// WhatsAppTemplate represents a pre-approved template for providers that require it.
type WhatsAppTemplate struct {
	Name     string
	Language string
	Params   []string
}
