package routes

import (
	"net/http"

	"clinic-backend/internal/ai_core"
	"clinic-backend/internal/appointment"
	"clinic-backend/internal/attachment"
	"clinic-backend/internal/auth"
	"clinic-backend/internal/document"
	"clinic-backend/internal/availability"
	"clinic-backend/internal/doctor"
	"clinic-backend/internal/doctor_dashboard"
	"clinic-backend/internal/followup"
	"clinic-backend/internal/inventory"
	"clinic-backend/internal/invoice"
	"clinic-backend/internal/medical"
	"clinic-backend/internal/notification"
	"clinic-backend/internal/ops_intelligence"
	"clinic-backend/internal/patient"
	"clinic-backend/internal/patientprofile"
	"clinic-backend/internal/procedurecatalog"
	"clinic-backend/internal/rating"
	"clinic-backend/internal/recurrence"
	"clinic-backend/internal/report"
	"clinic-backend/internal/reportai"
	"clinic-backend/internal/scheduling"
	"clinic-backend/internal/search"
	"clinic-backend/internal/settings"
	"clinic-backend/internal/tenant"
	"clinic-backend/internal/timeline"
	"clinic-backend/internal/upload"
	"clinic-backend/internal/visit"
	"clinic-backend/internal/whatsappbot"
)

type Handlers struct {
	AuthHandler         *auth.AuthHandler
	TenantHandler       *tenant.TenantHandler
	PatientHandler      *patient.PatientHandler
	AppointmentHandler  *appointment.AppointmentHandler
	DoctorHandler       *doctor.DoctorHandler
	UploadHandler       *upload.UploadHandler
	NotificationHandler *notification.NotificationHandler
	ReportHandler       *report.ReportHandler
	VisitHandler        *visit.VisitHandler
	InvoiceHandler      *invoice.InvoiceHandler
	MedicalHandler      *medical.MedicalHandler
	AttachmentHandler   *attachment.AttachmentHandler
	ReportAIHandler     *reportai.ReportAIHandler
	SearchHandler       *search.SearchHandler
	BotHandler          *whatsappbot.BotHandler
	RatingHandler       *rating.Handler
	DashHandler         *doctor_dashboard.Handler
	PPHandler           *patientprofile.PatientProfileHandler
	OpsHandler          *ops_intelligence.Handler
	TimelineHandler     *timeline.TimelineHandler
	AvailabilityHandler *availability.AvailabilityHandler
	SchedulingHandler   *scheduling.SmartSchedulingHandler
	SettingsHandler     *settings.Handler
	AIHandler           *ai_core.AIHandler
	FollowupHandler     *followup.Handler
	RecurrenceHandler   *recurrence.RecurrenceHandler
	InventoryHandler    *inventory.Handler
	ProcedureHandler    *procedurecatalog.Handler
	DocumentHandler     *document.DocumentHandler
}

func RegisterAll(mux *http.ServeMux, h Handlers) {
	registerPublicRoutes(mux, h)
	registerPatientRoutes(mux, h)
	registerDoctorRoutes(mux, h)
	registerAppointmentRoutes(mux, h)
	registerOpsAndSettingsRoutes(mux, h)
	registerFollowupUploadRoutes(mux, h)
	registerNotificationAttachmentRoutes(mux, h)
	registerBusinessRoutes(mux, h)
	registerDocumentRoutes(mux, h)
}
