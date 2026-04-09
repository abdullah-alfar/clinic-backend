package main

import (
	"fmt"
	"log"
	"net/http"

	"clinic-backend/internal/ai_core"
	"clinic-backend/internal/appointment"
	"clinic-backend/internal/attachment"
	"clinic-backend/internal/audit"
	"clinic-backend/internal/auth"
	"clinic-backend/internal/availability"
	"clinic-backend/internal/config"
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
	"clinic-backend/internal/platform/db"
	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/platform/http/routes"
	"clinic-backend/internal/procedurecatalog"
	"clinic-backend/internal/queue"
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
	"clinic-backend/internal/whatsapp"
	"clinic-backend/internal/whatsappbot"
	"github.com/joho/godotenv"
	"time"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not loaded, using system environment")
	}
	_ = godotenv.Load()

	database, err := db.NewDB()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	queueClient, err := queue.NewQueueClient("localhost:6379")
	if err != nil {
		log.Printf("Warning: Redis unavailable: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	waSender, err := whatsapp.NewSender(whatsapp.SenderFactoryConfig{
		Provider:         cfg.WhatsApp.Provider,
		TwilioAccountSID: cfg.WhatsApp.TwilioAccountSID,
		TwilioAuthToken:  cfg.WhatsApp.TwilioAuthToken,
		TwilioFrom:       cfg.WhatsApp.TwilioFrom,
	})
	if err != nil {
		log.Fatalf("failed to init whatsapp sender: %v", err)
	}
	auditSvc := audit.NewAuditService(database)
	authSvc := auth.NewAuthService(database)
	tenantSvc := tenant.NewTenantService(database)
	patientSvc := patient.NewPatientService(database, auditSvc)

	notifRepo := notification.NewPostgresNotificationRepository(database)
	prefSvc := notification.NewPreferenceService(notifRepo)
	notifDispatcher := notification.NewNotificationDispatcher(notifRepo, prefSvc, queueClient)

	availRepo := availability.NewPostgresAvailabilityRepository(database)
	advAvailSvc := availability.NewAvailabilityService(availRepo)
	advAvailHandler := availability.NewAvailabilityHandler(advAvailSvc)

	apptRepo := appointment.NewPostgresAppointmentRepository(database)
	apptSvc := appointment.NewAppointmentService(apptRepo, auditSvc, queueClient, notifDispatcher, advAvailSvc)
	doctorSvc := doctor.NewDoctorService(database, auditSvc)

	notifSvc := notification.NewNotificationService(database)
	reportSvc := report.NewReportService(database)

	inventoryRepo := inventory.NewPostgresRepository(database)
	inventorySvc := inventory.NewService(inventoryRepo)
	inventoryHandler := inventory.NewHandler(inventorySvc)

	procRepo := procedurecatalog.NewPostgresRepository(database)
	procSvc := procedurecatalog.NewService(procRepo)
	procHandler := procedurecatalog.NewHandler(procSvc)

	medicalRepo := medical.NewMedicalRepository(database)
	medicalSvc := medical.NewMedicalService(medicalRepo, auditSvc, inventoryRepo, procRepo)

	visitSvc := visit.NewVisitService(database, auditSvc, medicalSvc)

	invoiceRepo := invoice.NewPostgresInvoiceRepository(database)
	invoiceSvc := invoice.NewInvoiceService(invoiceRepo, database)

	schedulingSvc := scheduling.NewSmartSchedulingService(advAvailSvc)
	schedulingHandler := scheduling.NewSmartSchedulingHandler(schedulingSvc)

	followupRepo := followup.NewPostgresRepository(database)
	followupSvc := followup.NewService(followupRepo, notifDispatcher)
	followupHandler := followup.NewHandler(followupSvc)

	settingsRepo := settings.NewPostgresRepository(database)
	settingsSvc := settings.NewService(settingsRepo)
	settingsHandler := settings.NewHandler(settingsSvc)

	// Start follow-up scheduler
	followupScheduler := followup.NewScheduler(followupSvc, 1*time.Hour)
	followupScheduler.Start()

	recurrenceRepo := recurrence.NewPostgresRecurrenceRepository(database)
	recurrenceSvc := recurrence.NewRecurrenceService(recurrenceRepo, apptRepo, advAvailSvc)
	recurrenceHandler := recurrence.NewRecurrenceHandler(recurrenceSvc)

	attRepo := attachment.NewPostgresRepository(database)
	attStorage := attachment.NewLocalFileStorage("./uploads")
	attSvc := attachment.NewAttachmentService(attRepo, attStorage, auditSvc)

	aiRepo := reportai.NewPostgresRepository(database)
	aiProvider := reportai.NewMockAIProvider()
	aiSvc := reportai.NewReportAIService(aiRepo, aiProvider, auditSvc)

	providerRegistry := search.NewProviderRegistry()
	providerRegistry.Register(search.NewPatientProvider(database))
	providerRegistry.Register(search.NewDoctorProvider(database))
	providerRegistry.Register(search.NewAppointmentProvider(database))
	providerRegistry.Register(search.NewInvoiceProvider(database))
	providerRegistry.Register(search.NewAttachmentProvider(database))
	providerRegistry.Register(search.NewVisitNoteProvider(database))
	providerRegistry.Register(search.NewNotificationProvider(database))
	providerRegistry.Register(search.NewMemoryProvider(database))
	providerRegistry.Register(search.NewAuditProvider(database))
	providerRegistry.Register(search.NewScheduleProvider(database))

	searchSvc := search.NewSearchService(providerRegistry)

	// AI Core System Tools
	aiTools := ai_core.NewSystemTools()
	aiTools.Register(ai_core.NewGetAvailableSlotsTool(schedulingSvc))
	aiTools.Register(ai_core.NewSearchPatientsTool(searchSvc))
	aiTools.Register(ai_core.NewCreateAppointmentTool(apptSvc))
	aiTools.Register(ai_core.NewCancelAppointmentTool(apptSvc))

	aiMemoryManager := ai_core.NewTransientMemory(30 * time.Minute)
	aiCoreSvc := ai_core.NewAIService(aiTools, aiMemoryManager, settingsRepo)
	aiCoreHandler := ai_core.NewAIHandler(aiCoreSvc)

	// Bot depends on ai_core now
	botRepo := whatsappbot.NewPostgresBotRepository(database)
	botSvc := whatsappbot.NewBotService(botRepo, waSender, apptSvc, advAvailSvc, doctorSvc, aiCoreSvc)

	// Handlers
	authHandler := auth.NewAuthHandler(authSvc)
	tenantHandler := tenant.NewTenantHandler(tenantSvc)
	patientHandler := patient.NewPatientHandler(patientSvc)
	apptHandler := appointment.NewAppointmentHandler(apptSvc, advAvailSvc, doctorSvc)
	doctorHandler := doctor.NewDoctorHandler(doctorSvc)
	uploadHandler := upload.NewUploadHandler(database, auditSvc)
	notifHandler := notification.NewNotificationHandler(notifSvc, notifRepo, prefSvc)
	reportHandler := report.NewReportHandler(reportSvc)
	visitHandler := visit.NewVisitHandler(visitSvc)
	invoiceHandler := invoice.NewInvoiceHandler(invoiceSvc)
	medicalHandler := medical.NewMedicalHandler(medicalSvc)

	attHandler := attachment.NewAttachmentHandler(attSvc)
	aiHandler := reportai.NewReportAIHandler(aiSvc, attRepo)
	searchHandler := search.NewSearchHandler(searchSvc)
	botHandler := whatsappbot.NewBotHandler(botSvc, "dev_secret")

	// Rating & Feedback
	ratingRepo := rating.NewRepository(database)
	ratingSvc := rating.NewService(ratingRepo, apptRepo)
	ratingHandler := rating.NewHandler(ratingSvc)

	// Doctor Dashboard Initialization
	dashRepo := doctor_dashboard.NewRepository(database)
	dashSvc := doctor_dashboard.NewService(dashRepo)
	dashHandler := doctor_dashboard.NewHandler(dashSvc)

	// Patient Profile 360
	ppRepo := patientprofile.NewRepository(database)
	ppSvc := patientprofile.NewService(ppRepo)
	ppHandler := patientprofile.NewHandler(ppSvc)

	// Operational Intelligence
	opsRepo := ops_intelligence.NewPostgresRepository(database)
	opsPredictor := ops_intelligence.NewPredictor()
	opsAnalyzer := ops_intelligence.NewAnalyzer()
	opsInbox := ops_intelligence.NewInbox()
	opsSvc := ops_intelligence.NewService(opsRepo, opsPredictor, opsAnalyzer, opsInbox)
	opsHandler := ops_intelligence.NewHandler(opsSvc)

	// Timeline Aggregation
	timelineRepo := timeline.NewPostgresTimelineRepository(database)
	timelineSvc := timeline.NewTimelineService(timelineRepo)
	timelineHandler := timeline.NewTimelineHandler(timelineSvc)

	mux := http.NewServeMux()

	routes.RegisterAll(mux, routes.Handlers{
		AuthHandler:         authHandler,
		TenantHandler:       tenantHandler,
		PatientHandler:      patientHandler,
		AppointmentHandler:  apptHandler,
		DoctorHandler:       doctorHandler,
		UploadHandler:       uploadHandler,
		NotificationHandler: notifHandler,
		ReportHandler:       reportHandler,
		VisitHandler:        visitHandler,
		InvoiceHandler:      invoiceHandler,
		MedicalHandler:      medicalHandler,
		AttachmentHandler:   attHandler,
		ReportAIHandler:     aiHandler,
		SearchHandler:       searchHandler,
		BotHandler:          botHandler,
		RatingHandler:       ratingHandler,
		DashHandler:         dashHandler,
		PPHandler:           ppHandler,
		OpsHandler:          opsHandler,
		TimelineHandler:     timelineHandler,
		AvailabilityHandler: advAvailHandler,
		SchedulingHandler:   schedulingHandler,
		SettingsHandler:     settingsHandler,
		AIHandler:           aiCoreHandler,
		FollowupHandler:     followupHandler,
		RecurrenceHandler:   recurrenceHandler,
		InventoryHandler:    inventoryHandler,
		ProcedureHandler:    procHandler,
	})
	fmt.Println("Starting Clinic SaaS Phase 3 API on :8080")
	if err := http.ListenAndServe(":8080", myhttp.CORSMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}
