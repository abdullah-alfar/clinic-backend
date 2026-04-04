package report

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"clinic-backend/internal/appointment"
	"clinic-backend/internal/patient"
)

type DashboardSummary struct {
	TotalPatients          int `json:"total_patients"`
	TotalDoctors           int `json:"total_doctors"`
	AppointmentsToday      int `json:"appointments_today"`
	UpcomingAppointments   int `json:"upcoming_appointments"`
	CompletedAppointments  int `json:"completed_appointments"`
	CanceledAppointments   int `json:"canceled_appointments"`
	NoShowCount            int `json:"no_show_count"`
}

type ReportService struct {
	db *sql.DB
}

func NewReportService(db *sql.DB) *ReportService {
	return &ReportService{db: db}
}

func (s *ReportService) GetSummary(tenantID uuid.UUID) (*DashboardSummary, error) {
	var summary DashboardSummary

	// Patients count
	s.db.QueryRow("SELECT COUNT(1) FROM patients WHERE tenant_id = $1", tenantID).Scan(&summary.TotalPatients)
	
	// Doctors count
	s.db.QueryRow("SELECT COUNT(1) FROM doctors WHERE tenant_id = $1", tenantID).Scan(&summary.TotalDoctors)

	// Appointments status breakdown
	todayStart := time.Now().Truncate(24 * time.Hour)
	todayEnd := todayStart.Add(24 * time.Hour)

	s.db.QueryRow("SELECT COUNT(1) FROM appointments WHERE tenant_id = $1 AND start_time >= $2 AND end_time < $3", tenantID, todayStart, todayEnd).Scan(&summary.AppointmentsToday)
	s.db.QueryRow("SELECT COUNT(1) FROM appointments WHERE tenant_id = $1 AND start_time >= $2 AND status = 'scheduled'", tenantID, time.Now()).Scan(&summary.UpcomingAppointments)
	s.db.QueryRow("SELECT COUNT(1) FROM appointments WHERE tenant_id = $1 AND status = 'completed'", tenantID).Scan(&summary.CompletedAppointments)
	s.db.QueryRow("SELECT COUNT(1) FROM appointments WHERE tenant_id = $1 AND status = 'canceled'", tenantID).Scan(&summary.CanceledAppointments)
	s.db.QueryRow("SELECT COUNT(1) FROM appointments WHERE tenant_id = $1 AND status = 'no_show'", tenantID).Scan(&summary.NoShowCount)

	return &summary, nil
}

func (s *ReportService) GetAppointmentsReport(tenantID uuid.UUID, doctorID *uuid.UUID, status string, dateFrom, dateTo *time.Time) ([]*appointment.Appointment, error) {
	query := `SELECT id, patient_id, doctor_id, status, start_time, end_time FROM appointments WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	idx := 2

	if doctorID != nil {
		query += ` AND doctor_id = $` + string(rune(idx+'0'))
		args = append(args, *doctorID)
		idx++
	}
	if status != "" {
		query += ` AND status = $` + string(rune(idx+'0'))
		args = append(args, status)
		idx++
	}
	if dateFrom != nil {
		query += ` AND start_time >= $` + string(rune(idx+'0'))
		args = append(args, *dateFrom)
		idx++
	}
	if dateTo != nil {
		query += ` AND end_time <= $` + string(rune(idx+'0'))
		args = append(args, *dateTo)
		idx++
	}

	query += ` ORDER BY start_time DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*appointment.Appointment
	for rows.Next() {
		var a appointment.Appointment
		if err := rows.Scan(&a.ID, &a.PatientID, &a.DoctorID, &a.Status, &a.StartTime, &a.EndTime); err == nil {
			a.TenantID = tenantID
			list = append(list, &a)
		}
	}
	return list, nil
}

func (s *ReportService) GetPatientsReport(tenantID uuid.UUID, dateFrom, dateTo *time.Time) ([]*patient.Patient, error) {
	query := `SELECT id, first_name, last_name, email, created_at FROM patients WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	idx := 2

	if dateFrom != nil {
		query += ` AND created_at >= $` + string(rune(idx+'0'))
		args = append(args, *dateFrom)
		idx++
	}
	if dateTo != nil {
		query += ` AND created_at <= $` + string(rune(idx+'0'))
		args = append(args, *dateTo)
		idx++
	}

	query += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*patient.Patient
	for rows.Next() {
		var p patient.Patient
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Email, &p.CreatedAt); err == nil {
			list = append(list, &p)
		}
	}
	return list, nil
}
