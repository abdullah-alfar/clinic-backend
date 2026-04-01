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
	"clinic-backend/internal/notification"
	"clinic-backend/internal/patient"
	"clinic-backend/internal/platform/db"
	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/queue"
	"clinic-backend/internal/report"
	"clinic-backend/internal/reportai"
	"clinic-backend/internal/tenant"
	"clinic-backend/internal/upload"
	"clinic-backend/internal/visit"
	"clinic-backend/internal/search"
)

func main() {
	database, err := db.NewPostgresDB("localhost", "5432", "postgres", "root", "clinic")
	if err != nil {
		log.Printf("Warning: Failed to connect to DB: %v", err)
	}

	// Setup Redis Queue
	qClient, err := queue.NewQueueClient("localhost:6379")
	if err != nil {
		log.Printf("Warning: Redis unavailable: %v", err)
	}

	// Services
	auditSvc := audit.NewAuditService(database)
	authSvc := auth.NewAuthService(database)
	tenantSvc := tenant.NewTenantService(database)
	patientSvc := patient.NewPatientService(database, auditSvc)
	apptRepo := appointment.NewPostgresAppointmentRepository(database)
	apptSvc := appointment.NewAppointmentService(apptRepo, auditSvc, qClient)
	availSvc := appointment.NewAvailabilityService(apptRepo)
	doctorSvc := doctor.NewDoctorService(database, auditSvc)

	// Advanced Availability
	availRepo := availability.NewPostgresAvailabilityRepository(database)
	advAvailSvc := availability.NewAvailabilityService(availRepo)
	advAvailHandler := availability.NewAvailabilityHandler(advAvailSvc)
	notifSvc := notification.NewNotificationService(database)
	reportSvc := report.NewReportService(database)
	visitSvc := visit.NewVisitService(database, auditSvc)

	invRepo := invoice.NewPostgresInvoiceRepository(database)
	invSvc := invoice.NewInvoiceService(invRepo, database)

	attRepo := attachment.NewPostgresRepository(database)
	attStorage := attachment.NewLocalFileStorage("./uploads")
	attSvc := attachment.NewAttachmentService(attRepo, attStorage, auditSvc)

	aiRepo := reportai.NewPostgresRepository(database)
	aiProvider := reportai.NewMockAIProvider()
	aiSvc := reportai.NewReportAIService(aiRepo, aiProvider, auditSvc)

	searchRepo := search.NewPostgresSearchRepository(database)
	searchSvc := search.NewSearchService(searchRepo)

	// Handlers
	authHandler := auth.NewAuthHandler(authSvc)
	tenantHandler := tenant.NewTenantHandler(tenantSvc)
	patientHandler := patient.NewPatientHandler(patientSvc)
	apptHandler := appointment.NewAppointmentHandler(apptSvc, availSvc)
	doctorHandler := doctor.NewDoctorHandler(doctorSvc)
	uploadHandler := upload.NewUploadHandler(database, auditSvc)
	notifHandler := notification.NewNotificationHandler(notifSvc)
	reportHandler := report.NewReportHandler(reportSvc)
	visitHandler := visit.NewVisitHandler(visitSvc)
	invHandler := invoice.NewInvoiceHandler(invSvc)

	attHandler := attachment.NewAttachmentHandler(attSvc)
	aiHandler := reportai.NewReportAIHandler(aiSvc, attRepo)
	searchHandler := search.NewSearchHandler(searchSvc)

	mux := http.NewServeMux()

	// Public Routes
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.HandleRefresh)
	mux.HandleFunc("GET /api/v1/tenants/config", tenantHandler.HandleGetConfig)

	// Global Search
	mux.Handle("GET /api/v1/search", myhttp.AuthMiddleware(http.HandlerFunc(searchHandler.HandleSearch)))

	// Protected Auth Route
	mux.Handle("GET /api/v1/auth/me", myhttp.AuthMiddleware(http.HandlerFunc(authHandler.HandleMe)))

	// Patients RBAC
	mux.Handle("GET /api/v1/patients", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(patientHandler.HandlePatients))))
	mux.Handle("POST /api/v1/patients", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(patientHandler.HandlePatients))))
	mux.Handle("GET /api/v1/patients/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(patientHandler.HandlePatientByID))))

	// Visits & Timeline
	mux.Handle("POST /api/v1/visits", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(visitHandler.HandleVisits))))
	mux.Handle("GET /api/v1/patients/{id}/timeline", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(visitHandler.HandlePatientTimeline))))

	// Doctors RBAC
	mux.Handle("GET /api/v1/doctors", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(doctorHandler.ServeHTTP))))
	mux.Handle("POST /api/v1/doctors", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(doctorHandler.ServeHTTP))))
	mux.Handle("PATCH /api/v1/doctors/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(doctorHandler.ServeHTTP))))
	mux.Handle("DELETE /api/v1/doctors/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(doctorHandler.ServeHTTP))))

	// ── Advanced Doctor Availability ─────────────────────────────────────────
	// Full config view (schedules + breaks + exceptions)
	mux.Handle("GET /api/v1/doctors/{id}/availability",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor", "receptionist")(http.HandlerFunc(advAvailHandler.HandleGetFullAvailability))))

	// Computed slot list
	mux.Handle("GET /api/v1/doctors/{id}/availability/slots",
		myhttp.AuthMiddleware(http.HandlerFunc(advAvailHandler.HandleGetSlots)))

	// Schedule CRUD
	mux.Handle("POST /api/v1/doctors/{id}/availability/schedules",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleCreateSchedule))))
	mux.Handle("PATCH /api/v1/doctors/{id}/availability/schedules/{sid}",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleUpdateSchedule))))
	mux.Handle("DELETE /api/v1/doctors/{id}/availability/schedules/{sid}",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleDeleteSchedule))))

	// Break CRUD
	mux.Handle("POST /api/v1/doctors/{id}/availability/schedules/{sid}/breaks",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleCreateBreak))))
	mux.Handle("DELETE /api/v1/doctors/{id}/availability/breaks/{bid}",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleDeleteBreak))))

	// Exception CRUD
	mux.Handle("GET /api/v1/doctors/{id}/availability/exceptions",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor", "receptionist")(http.HandlerFunc(advAvailHandler.HandleListExceptions))))
	mux.Handle("POST /api/v1/doctors/{id}/availability/exceptions",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleCreateException))))
	mux.Handle("DELETE /api/v1/doctors/{id}/availability/exceptions/{eid}",
		myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(http.HandlerFunc(advAvailHandler.HandleDeleteException))))

	// Appointments Read
	mux.Handle("GET /api/v1/appointments/availability", myhttp.AuthMiddleware(http.HandlerFunc(apptHandler.HandleGetAvailability)))
	mux.Handle("GET /api/v1/appointments/next-available", myhttp.AuthMiddleware(http.HandlerFunc(apptHandler.HandleGetNextAvailable)))

	// Calendar view — returns enriched appointments with patient/doctor names
	mux.Handle("GET /api/v1/appointments/calendar", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(apptHandler.HandleGetCalendar))))

	// Appointments RBAC (Create)
	mux.Handle("POST /api/v1/appointments", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "patient")(http.HandlerFunc(apptHandler.HandleSchedule))))

	// Appointments RBAC (Update / Reschedule / Cancel)
	mux.Handle("PATCH /api/v1/appointments/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(apptHandler.HandleUpdate))))
	mux.Handle("PATCH /api/v1/appointments/{id}/reschedule", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(apptHandler.HandleReschedule))))
	mux.Handle("PATCH /api/v1/appointments/{id}/cancel", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(apptHandler.HandleStatus("canceled"))))

	// Appointments RBAC (Confirm / Complete)
	mux.Handle("PATCH /api/v1/appointments/{id}/confirm", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(apptHandler.HandleStatus("confirmed"))))
	mux.Handle("PATCH /api/v1/appointments/{id}/complete", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "doctor")(apptHandler.HandleStatus("completed"))))

	// Uploads & Static files
	mux.Handle("POST /api/v1/uploads", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(uploadHandler.HandleUpload))))
	mux.Handle("GET /uploads/{tenant_id}/{file}", myhttp.AuthMiddleware(http.HandlerFunc(uploadHandler.HandleStatic)))

	// Notifications
	mux.Handle("GET /api/v1/notifications", myhttp.AuthMiddleware(http.HandlerFunc(notifHandler.HandleList)))
	mux.Handle("PATCH /api/v1/notifications/{id}/read", myhttp.AuthMiddleware(http.HandlerFunc(notifHandler.HandleRead)))

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
