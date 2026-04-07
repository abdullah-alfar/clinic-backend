package main

import (
	"fmt"
	"log"
	"net/http"

	"clinic-backend/internal/appointment"
	"clinic-backend/internal/attachment"
	"clinic-backend/internal/audit"
	"clinic-backend/internal/auth"
	"clinic-backend/internal/availability"
	"clinic-backend/internal/config"
	"clinic-backend/internal/doctor"
	"clinic-backend/internal/ai_core"
	"clinic-backend/internal/doctor_dashboard"
	"clinic-backend/internal/inventory"
	"clinic-backend/internal/invoice"
	"clinic-backend/internal/medical"
	"clinic-backend/internal/notification"
	"clinic-backend/internal/ops_intelligence"
	"clinic-backend/internal/patient"
	"clinic-backend/internal/patientprofile"
	"clinic-backend/internal/platform/db"
	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/procedurecatalog"
	"clinic-backend/internal/queue"
	"clinic-backend/internal/recurrence"
	"clinic-backend/internal/rating"
	"clinic-backend/internal/report"
	"clinic-backend/internal/reportai"
	"clinic-backend/internal/scheduling"
	"clinic-backend/internal/search"
	"clinic-backend/internal/settings"
	"clinic-backend/internal/tenant"
	"clinic-backend/internal/timeline"
	"clinic-backend/internal/upload"
	"clinic-backend/internal/visit"
	"clinic-backend/internal/followup"
	"clinic-backend/internal/whatsapp"
	"clinic-backend/internal/whatsappbot"
	"github.com/joho/godotenv"
	"time"
)

func main() {
	database, err := db.NewPostgresDB("localhost", "5432", "postgres", "root", "clinic")
	if err != nil {
		log.Printf("Warning: Failed to connect to DB: %v", err)
	}

	queueClient, err := queue.NewQueueClient("localhost:6379")
	if err != nil {
		log.Printf("Warning: Redis unavailable: %v", err)
	}

	if err := godotenv.Load(); err != nil {
		log.Println(".env file not loaded, using system environment")
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

	// Public Routes
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.HandleRefresh)
	mux.HandleFunc("GET /api/v1/tenants/config", tenantHandler.HandleGetConfig)

	// Webhooks
	mux.HandleFunc("POST /webhooks/whatsapp", botHandler.HandleWebhook)

	// Global Search
	mux.Handle("GET /api/v1/search", myhttp.AuthMiddleware(http.HandlerFunc(searchHandler.HandleSearch)))

	// Protected Auth Route
	mux.Handle("GET /api/v1/auth/me", myhttp.AuthMiddleware(http.HandlerFunc(authHandler.HandleMe)))

	// Doctor Dashboard Route
	mux.Handle("GET /api/v1/doctor-dashboard", myhttp.AuthMiddleware(http.HandlerFunc(dashHandler.GetDashboard)))

	// Patients RBAC
	mux.Handle("GET /api/v1/patients", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(patientHandler.HandlePatients))))
	mux.Handle("POST /api/v1/patients", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(patientHandler.HandlePatients))))
	mux.Handle("GET /api/v1/patients/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(patientHandler.HandlePatientByID))))
	mux.Handle("GET /api/v1/patients/{id}/profile", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(ppHandler.GetProfile))))
	mux.Handle("GET /api/v1/patients/{id}/activities", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(ppHandler.GetActivityStream))))

	mux.Handle("PUT /api/v1/patients/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(patientHandler.HandleUpdatePatient))))
	mux.Handle("DELETE /api/v1/patients/{id}",
		myhttp.AuthMiddleware(
			myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(patientHandler.HandleDeletePatient)),
		),
	)

	// Visits & Timeline
	mux.Handle("POST /api/v1/visits", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(visitHandler.HandleVisits))))
	mux.Handle("GET /api/v1/patients/{id}/timeline", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(timelineHandler.HandlePatientTimeline))))

	// Medical Records
	mux.Handle("GET /api/v1/patients/{id}/medical-records", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor", "receptionist")(http.HandlerFunc(medicalHandler.ListRecords))))
	mux.Handle("POST /api/v1/patients/{id}/medical-records", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(medicalHandler.CreateRecord))))
	mux.Handle("GET /api/v1/medical-records/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor", "receptionist")(http.HandlerFunc(medicalHandler.GetRecord))))
	mux.Handle("PATCH /api/v1/medical-records/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(medicalHandler.UpdateRecord))))
	mux.Handle("DELETE /api/v1/medical-records/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(medicalHandler.DeleteRecord))))
	mux.Handle("POST /api/v1/medical-records/{id}/procedures", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(medicalHandler.HandleAddProcedure))))

	// Doctors RBAC
	mux.Handle("GET /api/v1/doctors", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(doctorHandler.ServeHTTP))))
	mux.Handle("POST /api/v1/doctors", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(doctorHandler.ServeHTTP))))
	mux.Handle("PUT /api/v1/doctors/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(doctorHandler.ServeHTTP))))
	mux.Handle("DELETE /api/v1/doctors/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(doctorHandler.ServeHTTP))))

	// ── Advanced Doctor Availability ─────────────────────────────────────────
	mux.Handle("GET /api/v1/doctors/{id}/availability",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor", "receptionist")(http.HandlerFunc(advAvailHandler.HandleGetFullAvailability))))
	mux.Handle("GET /api/v1/doctors/{id}/availability/slots",
		myhttp.AuthMiddleware(http.HandlerFunc(advAvailHandler.HandleGetSlots)))
	mux.Handle("POST /api/v1/doctors/{id}/availability/schedules",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleCreateSchedule))))
	mux.Handle("PATCH /api/v1/doctors/{id}/availability/schedules/{sid}",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleUpdateSchedule))))
	mux.Handle("DELETE /api/v1/doctors/{id}/availability/schedules/{sid}",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleDeleteSchedule))))
	mux.Handle("POST /api/v1/doctors/{id}/availability/schedules/{sid}/breaks",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleCreateBreak))))
	mux.Handle("DELETE /api/v1/doctors/{id}/availability/breaks/{bid}",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleDeleteBreak))))
	mux.Handle("GET /api/v1/doctors/{id}/availability/exceptions",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor", "receptionist")(http.HandlerFunc(advAvailHandler.HandleListExceptions))))
	mux.Handle("POST /api/v1/doctors/{id}/availability/exceptions",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleCreateException))))
	mux.Handle("DELETE /api/v1/doctors/{id}/availability/exceptions/{eid}",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleDeleteException))))

	// Appointments Read
	mux.Handle("GET /api/v1/appointments/availability", myhttp.AuthMiddleware(http.HandlerFunc(apptHandler.HandleGetAvailability)))
	mux.Handle("GET /api/v1/appointments/next-available", myhttp.AuthMiddleware(http.HandlerFunc(apptHandler.HandleGetNextAvailable)))
	mux.Handle("GET /api/v1/appointments/calendar", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(apptHandler.HandleGetCalendar))))
	mux.Handle("GET /api/v1/appointments/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(apptHandler.HandleGetByID))))

	// Appointments RBAC
	mux.Handle("POST /api/v1/appointments", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "patient")(http.HandlerFunc(apptHandler.HandleSchedule))))
	mux.Handle("PATCH /api/v1/appointments/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(apptHandler.HandleUpdate))))
	mux.Handle("PATCH /api/v1/appointments/{id}/reschedule", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(apptHandler.HandleReschedule))))
	mux.Handle("PATCH /api/v1/appointments/{id}/cancel", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(apptHandler.HandleStatus("canceled"))))
	mux.Handle("PATCH /api/v1/appointments/{id}/confirm", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(apptHandler.HandleStatus("confirmed"))))
	mux.Handle("PATCH /api/v1/appointments/{id}/complete", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(apptHandler.HandleStatus("completed"))))
	mux.Handle("PATCH /api/v1/appointments/{id}/no-show", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(apptHandler.HandleStatus("no_show"))))

	// Operational Intelligence
	mux.Handle("GET /api/v1/appointments/{id}/no-show-risk", myhttp.AuthMiddleware(http.HandlerFunc(opsHandler.HandleNoShowRisk)))
	mux.Handle("GET /api/v1/revenue/missing", myhttp.AuthMiddleware(http.HandlerFunc(opsHandler.HandleMissingRevenue)))
	mux.Handle("GET /api/v1/communications", myhttp.AuthMiddleware(http.HandlerFunc(opsHandler.HandleCommunications)))

	// Smart Scheduling
	mux.Handle("GET /api/v1/appointments/smart-suggestions", myhttp.AuthMiddleware(http.HandlerFunc(schedulingHandler.HandleSmartSuggestions)))

	// Settings (System Control Panel)
	mux.Handle("GET /api/v1/settings", myhttp.AuthMiddleware(http.HandlerFunc(settingsHandler.HandleGetSettings)))
	mux.Handle("PUT /api/v1/settings", myhttp.AuthMiddleware(http.HandlerFunc(settingsHandler.HandleUpdateSettings)))
	mux.Handle("POST /api/v1/settings/test-ai", myhttp.AuthMiddleware(http.HandlerFunc(settingsHandler.HandleTestAI)))
	mux.Handle("POST /api/v1/settings/test-email", myhttp.AuthMiddleware(http.HandlerFunc(settingsHandler.HandleTestEmail)))
	mux.Handle("POST /api/v1/settings/test-whatsapp", myhttp.AuthMiddleware(http.HandlerFunc(settingsHandler.HandleTestWhatsApp)))

	// AI Core
	mux.Handle("POST /api/v1/ai/chat", myhttp.AuthMiddleware(http.HandlerFunc(aiCoreHandler.HandleChat)))

	// Follow-ups
	mux.Handle("POST /api/v1/follow-ups", myhttp.AuthMiddleware(http.HandlerFunc(followupHandler.HandleCreate)))
	mux.Handle("GET /api/v1/follow-ups", myhttp.AuthMiddleware(http.HandlerFunc(followupHandler.HandleList)))
	mux.Handle("PATCH /api/v1/follow-ups/", myhttp.AuthMiddleware(http.HandlerFunc(followupHandler.HandleUpdateStatus))) // Note: path matching logic in handler
	mux.Handle("GET /api/v1/patients/{id}/follow-ups", myhttp.AuthMiddleware(http.HandlerFunc(followupHandler.HandlePatientFollowUps)))

	// Recurring Appointments
	mux.Handle("POST /api/v1/appointments/recurring", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(recurrenceHandler.CreateRule))))
	mux.Handle("GET /api/v1/appointments/recurring", myhttp.AuthMiddleware(http.HandlerFunc(recurrenceHandler.ListRules)))

	// Uploads
	mux.Handle("POST /api/v1/uploads", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(uploadHandler.HandleUpload))))
	mux.Handle("GET /uploads/{tenant_id}/{file}", myhttp.AuthMiddleware(http.HandlerFunc(uploadHandler.HandleStatic)))

	// Notifications & WhatsApp Bot
	mux.Handle("GET /api/v1/notifications", myhttp.AuthMiddleware(http.HandlerFunc(notifHandler.HandleList)))
	mux.Handle("PATCH /api/v1/notifications/{id}/read", myhttp.AuthMiddleware(http.HandlerFunc(notifHandler.HandleRead)))
	mux.Handle("GET /api/v1/patients/{id}/notifications", myhttp.AuthMiddleware(http.HandlerFunc(notifHandler.HandlePatientHistory)))
	mux.Handle("GET /api/v1/patients/{id}/notification-preferences", myhttp.AuthMiddleware(http.HandlerFunc(notifHandler.HandleGetPreferences)))
	mux.Handle("PUT /api/v1/patients/{id}/notification-preferences", myhttp.AuthMiddleware(http.HandlerFunc(notifHandler.HandleUpdatePreferences)))
	mux.Handle("GET /api/v1/patients/{id}/whatsapp/history", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor", "receptionist")(http.HandlerFunc(botHandler.HandlePatientHistory))))
	mux.Handle("GET /api/v1/patients/{id}/whatsapp/status", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor", "receptionist")(http.HandlerFunc(botHandler.HandleBotStatus))))

	// Attachments
	mux.Handle("POST /api/v1/patients/{id}/attachments", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(attHandler.HandleUploadAttachment))))
	mux.Handle("GET /api/v1/patients/{id}/attachments", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(attHandler.HandleListAttachments))))
	mux.Handle("GET /api/v1/attachments/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(attHandler.HandleGetAttachment))))
	mux.Handle("DELETE /api/v1/attachments/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(attHandler.HandleDeleteAttachment))))

	// Report AI
	mux.Handle("POST /api/v1/attachments/{id}/analyze", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(aiHandler.HandleAnalyzeReport))))
	mux.Handle("GET /api/v1/attachments/{id}/analyses", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(aiHandler.HandleGetAnalyses))))

	// Reporting & Dashboards
	mux.Handle("GET /api/v1/dashboard/summary", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(reportHandler.HandleDashboard))))
	mux.Handle("GET /api/v1/reports/appointments", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(reportHandler.HandleAppointmentsReport))))
	mux.Handle("GET /api/v1/reports/patients", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(reportHandler.HandlePatientsReport))))

	// Invoices
	mux.Handle("POST /api/v1/invoices", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(invoiceHandler.HandleCreate))))
	mux.Handle("GET /api/v1/patients/{id}/invoices", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(invoiceHandler.HandleListPatientInvoices))))
	mux.Handle("PATCH /api/v1/invoices/{id}/pay", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(invoiceHandler.HandleMarkPaid))))

	// Inventory
	mux.Handle("GET /api/v1/inventory/items", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(inventoryHandler.HandleListItems))))
	mux.Handle("GET /api/v1/inventory/items/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(inventoryHandler.HandleGetItem))))
	mux.Handle("POST /api/v1/inventory/items", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(inventoryHandler.HandleCreateItem))))
	mux.Handle("PATCH /api/v1/inventory/items/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(inventoryHandler.HandleUpdateItem))))
	mux.Handle("POST /api/v1/inventory/items/{id}/adjust", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(inventoryHandler.HandleAdjustStock))))
	mux.Handle("GET /api/v1/inventory/items/{id}/movements", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(inventoryHandler.HandleListMovements))))

	// Procedure Catalog
	mux.Handle("GET /api/v1/procedures", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(procHandler.HandleListProcedures))))
	mux.Handle("GET /api/v1/procedures/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(procHandler.HandleGetProcedure))))
	mux.Handle("POST /api/v1/procedures", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(procHandler.HandleCreateProcedure))))
	mux.Handle("PATCH /api/v1/procedures/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(procHandler.HandleUpdateProcedure))))

	// Ratings
	mux.Handle("POST /api/v1/appointments/{id}/rating", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "patient")(http.HandlerFunc(ratingHandler.HandleSubmitRating))))
	mux.Handle("GET /api/v1/doctors/{id}/ratings", myhttp.AuthMiddleware(http.HandlerFunc(ratingHandler.HandleGetDoctorRatings)))
	mux.Handle("GET /api/v1/patients/{id}/ratings", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor", "receptionist")(http.HandlerFunc(ratingHandler.HandleGetPatientRatings))))
	mux.Handle("GET /api/v1/ratings/analytics", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(ratingHandler.HandleGetGlobalAnalytics))))

	// CORS wrapper
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	fmt.Println("Starting Clinic SaaS Phase 3 API on :8080")
	if err := http.ListenAndServe(":8080", corsMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}
