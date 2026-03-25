package main

import (
	"fmt"
	"log"
	"net/http"

	"clinic-backend/internal/appointment"
	"clinic-backend/internal/audit"
	"clinic-backend/internal/auth"
	"clinic-backend/internal/doctor"
	"clinic-backend/internal/notification"
	"clinic-backend/internal/patient"
	"clinic-backend/internal/platform/db"
	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/queue"
	"clinic-backend/internal/report"
	"clinic-backend/internal/tenant"
	"clinic-backend/internal/upload"
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
	notifSvc := notification.NewNotificationService(database)
	reportSvc := report.NewReportService(database)

	// Handlers
	authHandler := auth.NewAuthHandler(authSvc)
	tenantHandler := tenant.NewTenantHandler(tenantSvc)
	patientHandler := patient.NewPatientHandler(patientSvc)
	apptHandler := appointment.NewAppointmentHandler(apptSvc, availSvc)
	doctorHandler := doctor.NewDoctorHandler(doctorSvc)
	uploadHandler := upload.NewUploadHandler(database, auditSvc)
	notifHandler := notification.NewNotificationHandler(notifSvc)
	reportHandler := report.NewReportHandler(reportSvc)

	mux := http.NewServeMux()

	// Public Routes
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.HandleRefresh)
	mux.HandleFunc("GET /api/v1/tenants/config", tenantHandler.HandleGetConfig)

	// Protected Auth Route
	mux.Handle("GET /api/v1/auth/me", myhttp.AuthMiddleware(http.HandlerFunc(authHandler.HandleMe)))

	// Patients RBAC
	mux.Handle("GET /api/v1/patients", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(patientHandler.HandlePatients))))
	mux.Handle("POST /api/v1/patients", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(patientHandler.HandlePatients))))
	mux.Handle("GET /api/v1/patients/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(patientHandler.HandlePatientByID))))

	// Doctors RBAC
	mux.Handle("GET /api/v1/doctors", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(doctorHandler.ServeHTTP))))
	mux.Handle("POST /api/v1/doctors", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(doctorHandler.ServeHTTP))))
	mux.Handle("PATCH /api/v1/doctors/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(doctorHandler.ServeHTTP))))
	mux.Handle("DELETE /api/v1/doctors/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin")(http.HandlerFunc(doctorHandler.ServeHTTP))))

	// Appointments Read
	mux.Handle("GET /api/v1/appointments/availability", myhttp.AuthMiddleware(http.HandlerFunc(apptHandler.HandleGetAvailability)))
	mux.Handle("GET /api/v1/appointments/next-available", myhttp.AuthMiddleware(http.HandlerFunc(apptHandler.HandleGetNextAvailable)))

	// Appointments RBAC (Create/Read)
	mux.Handle("POST /api/v1/appointments", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "patient")(http.HandlerFunc(apptHandler.HandleSchedule))))

	// Appointments RBAC (Update / Cancel)
	mux.Handle("PATCH /api/v1/appointments/{id}", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist")(http.HandlerFunc(apptHandler.HandleUpdate))))
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

	// Reporting & Dashboards
	mux.Handle("GET /api/v1/dashboard/summary", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(reportHandler.HandleDashboard))))
	mux.Handle("GET /api/v1/reports/appointments", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(reportHandler.HandleAppointmentsReport))))
	mux.Handle("GET /api/v1/reports/patients", myhttp.AuthMiddleware(myhttp.RBACMiddleware("admin", "receptionist", "doctor")(http.HandlerFunc(reportHandler.HandlePatientsReport))))

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
