package patientprofile

import (
	"time"

	"github.com/google/uuid"
)

type PatientSummary struct {
	LastVisitAt             *time.Time `json:"last_visit_at"`
	UpcomingAppointmentAt   *time.Time `json:"upcoming_appointment_at"`
	PreferredDoctorID       *uuid.UUID `json:"preferred_doctor_id"`
	PreferredDoctorName     string     `json:"preferred_doctor_name"`
	TotalAppointments       int        `json:"total_appointments"`
	CompletedAppointments    int        `json:"completed_appointments"`
	CanceledAppointments     int        `json:"canceled_appointments"`
	NoShowCount             int        `json:"no_show_count"`
	TotalInvoices           int        `json:"total_invoices"`
	UnpaidInvoicesCount     int        `json:"unpaid_invoices_count"`
	AttachmentsCount        int        `json:"attachments_count"`
	MedicalRecordsCount     int        `json:"medical_records_count"`
	AverageRatingGiven      *float64   `json:"average_rating_given"`
}

type PatientFlag struct {
	Type  string `json:"type"`  // alert, medical, billing
	Label string `json:"label"`
}

type RecentActivity struct {
	Type      string    `json:"type"` // appointment, medical_record, report, invoice, communication
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Subtitle  string    `json:"subtitle"`
	Timestamp time.Time  `json:"timestamp"`
	Status    string    `json:"status"`
}
