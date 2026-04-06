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
	"clinic-backend/internal/doctor"
	"clinic-backend/internal/invoice"
	"clinic-backend/internal/medical"
	"clinic-backend/internal/notification"
	"clinic-backend/internal/patient"
	"clinic-backend/internal/platform/db"
	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/queue"
	"clinic-backend/internal/recurrence"
	"clinic-backend/internal/report"
	"clinic-backend/internal/reportai"
	"clinic-backend/internal/scheduling"
	"clinic-backend/internal/search"
	"clinic-backend/internal/tenant"
	"clinic-backend/internal/upload"
	"clinic-backend/internal/visit"
	"clinic-backend/internal/whatsapp"
	"clinic-backend/internal/whatsappbot"
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

	waSender := whatsapp.NewLogWhatsAppSender()

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

	botRepo := whatsappbot.NewPostgresBotRepository(database)
	botSvc := whatsappbot.NewBotService(botRepo, waSender, apptSvc, advAvailSvc, doctorSvc)

	notifSvc := notification.NewNotificationService(database)
	reportSvc := report.NewReportService(database)

	medicalRepo := medical.NewMedicalRepository(database)
	medicalSvc := medical.NewMedicalService(medicalRepo, auditSvc)

	visitSvc := visit.NewVisitService(database, auditSvc, medicalSvc)

	invRepo := invoice.NewPostgresInvoiceRepository(database)
	invSvc := invoice.NewInvoiceService(invRepo, database)

	schedulingSvc := scheduling.NewSmartSchedulingService(advAvailSvc)
	schedulingHandler := scheduling.NewSmartSchedulingHandler(schedulingSvc)

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
	invHandler := invoice.NewInvoiceHandler(invSvc)
	medicalHandler := medical.NewMedicalHandler(medicalSvc)

	attHandler := attachment.NewAttachmentHandler(attSvc)
	aiHandler := reportai.NewReportAIHandler(aiSvc, attRepo)
	searchHandler := search.NewSearchHandler(searchSvc)
	botHandler := whatsappbot.NewBotHandler(botSvc, "dev_secret")

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

	// Patients RBAC
	mux.Handle("GET /api/v1/patients", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(patientHandler.HandlePatients))))
	mux.Handle("POST /api/v1/patients", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(patientHandler.HandlePatients))))
	mux.Handle("GET /api/v1/patients/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(patientHandler.HandlePatientByID))))
	mux.Handle("PUT /api/v1/patients/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(patientHandler.HandleUpdatePatient))))
	mux.Handle("DELETE /api/v1/patients/{id}",
		myhttp.AuthMiddleware(
			myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(patientHandler.HandleDeletePatient)),
		),
	)

	// Visits & Timeline
	mux.Handle("POST /api/v1/visits", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(visitHandler.HandleVisits))))
	mux.Handle("GET /api/v1/patients/{id}/timeline", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(visitHandler.HandlePatientTimeline))))

	// Medical Records
	mux.Handle("GET /api/v1/patients/{id}/medical-records", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor", "receptionist")(http.HandlerFunc(medicalHandler.ListRecords))))
	mux.Handle("POST /api/v1/patients/{id}/medical-records", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(medicalHandler.CreateRecord))))
	mux.Handle("GET /api/v1/medical-records/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor", "receptionist")(http.HandlerFunc(medicalHandler.GetRecord))))
	mux.Handle("PATCH /api/v1/medical-records/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(medicalHandler.UpdateRecord))))
	mux.Handle("DELETE /api/v1/medical-records/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(medicalHandler.DeleteRecord))))

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

	// Appointments RBAC
	mux.Handle("POST /api/v1/appointments", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "patient")(http.HandlerFunc(apptHandler.HandleSchedule))))
	mux.Handle("PATCH /api/v1/appointments/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(apptHandler.HandleUpdate))))
	mux.Handle("PATCH /api/v1/appointments/{id}/reschedule", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(apptHandler.HandleReschedule))))
	mux.Handle("PATCH /api/v1/appointments/{id}/cancel", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(apptHandler.HandleStatus("canceled"))))
	mux.Handle("PATCH /api/v1/appointments/{id}/confirm", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(apptHandler.HandleStatus("confirmed"))))
	mux.Handle("PATCH /api/v1/appointments/{id}/complete", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(apptHandler.HandleStatus("completed"))))
	mux.Handle("PATCH /api/v1/appointments/{id}/no-show", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(apptHandler.HandleStatus("no_show"))))

	// Smart Scheduling
	mux.Handle("GET /api/v1/appointments/smart-suggestions", myhttp.AuthMiddleware(http.HandlerFunc(schedulingHandler.HandleSmartSuggestions)))

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
	mux.Handle("POST /api/v1/invoices", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(invHandler.HandleCreate))))
	mux.Handle("GET /api/v1/patients/{id}/invoices", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(invHandler.HandleListPatientInvoices))))
	mux.Handle("PATCH /api/v1/invoices/{id}/pay", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(invHandler.HandleMarkPaid))))

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
