package doctor_dashboard

import (
	"time"

	"github.com/google/uuid"
)

type DoctorSummary struct {
	ID        uuid.UUID `json:"id"`
	FullName  string    `json:"full_name"`
	Specialty string    `json:"specialty"`
}

type DashboardStats struct {
	AppointmentsToday   int `json:"appointments_today"`
	UpcomingTotal       int `json:"upcoming_total"`
	CompletedToday      int `json:"completed_today"`
	NoShowToday         int `json:"no_show_today"`
	PendingNotes        int `json:"pending_notes"`
	UnreadNotifications int `json:"unread_notifications"`
}

type AppointmentSummary struct {
	ID          uuid.UUID `json:"id"`
	PatientID   uuid.UUID `json:"patient_id"`
	PatientName string    `json:"patient_name"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Status      string    `json:"status"`
	Reason      string    `json:"reason"`
}

type RecentPatient struct {
	ID         uuid.UUID `json:"id"`
	FullName   string    `json:"full_name"`
	LastVisit  time.Time `json:"last_visit"`
	VisitNotes string    `json:"visit_notes"`
}

type MedicalActivity struct {
	ID           uuid.UUID `json:"id"`
	PatientID    uuid.UUID `json:"patient_id"`
	PatientName  string    `json:"patient_name"`
	Type         string    `json:"type"` // e.g., "visit", "record", "report"
	Description  string    `json:"description"`
	ActivityDate time.Time `json:"activity_date"`
}

type DashboardData struct {
	Doctor                 DoctorSummary        `json:"doctor"`
	Stats                  DashboardStats       `json:"stats"`
	TodayAppointments      []AppointmentSummary `json:"today_appointments"`
	UpcomingAppointments   []AppointmentSummary `json:"upcoming_appointments"`
	RecentPatients         []RecentPatient      `json:"recent_patients"`
	RecentMedicalActivity []MedicalActivity    `json:"recent_medical_activity"`
}
