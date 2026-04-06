package patientprofile

import (
	"database/sql"

	"clinic-backend/internal/patient"
	"github.com/google/uuid"
)

type PatientProfileRepository interface {
	GetPatient(tenantID, patientID uuid.UUID) (*patient.Patient, error)
	GetSummary(tenantID, patientID uuid.UUID) (PatientSummary, error)
	GetRecentAppointments(tenantID, patientID uuid.UUID, limit int) ([]RecentActivity, error)
	GetRecentMedicalRecords(tenantID, patientID uuid.UUID, limit int) ([]RecentActivity, error)
	GetRecentInvoices(tenantID, patientID uuid.UUID, limit int) ([]RecentActivity, error)
	GetRecentReports(tenantID, patientID uuid.UUID, limit int) ([]RecentActivity, error)
}

type postgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) PatientProfileRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetPatient(tenantID, patientID uuid.UUID) (*patient.Patient, error) {
	var p patient.Patient
	err := r.db.QueryRow(`
		SELECT id, first_name, last_name, phone, email, date_of_birth, gender, created_at
		FROM patients
		WHERE id = $1 AND tenant_id = $2
	`, patientID, tenantID).Scan(
		&p.ID, &p.FirstName, &p.LastName, &p.Phone, &p.Email, &p.DateOfBirth, &p.Gender, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *postgresRepository) GetSummary(tenantID, patientID uuid.UUID) (PatientSummary, error) {
	var s PatientSummary

	// 1. Appointment Counts
	err := r.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'canceled') as canceled,
			COUNT(*) FILTER (WHERE status = 'no_show') as no_show
		FROM appointments
		WHERE patient_id = $1 AND tenant_id = $2
	`, patientID, tenantID).Scan(&s.TotalAppointments, &s.CompletedAppointments, &s.CanceledAppointments, &s.NoShowCount)
	if err != nil && err != sql.ErrNoRows {
		return s, err
	}

	// 2. Last Visit
	err = r.db.QueryRow(`
		SELECT start_time 
		FROM appointments 
		WHERE patient_id = $1 AND tenant_id = $2 AND status = 'completed' AND start_time < NOW()
		ORDER BY start_time DESC LIMIT 1
	`, patientID, tenantID).Scan(&s.LastVisitAt)
	if err != nil && err != sql.ErrNoRows {
		return s, err
	}

	// 3. Upcoming Appointment
	err = r.db.QueryRow(`
		SELECT start_time 
		FROM appointments 
		WHERE patient_id = $1 AND tenant_id = $2 AND status IN ('scheduled', 'confirmed') AND start_time > NOW()
		ORDER BY start_time ASC LIMIT 1
	`, patientID, tenantID).Scan(&s.UpcomingAppointmentAt)
	if err != nil && err != sql.ErrNoRows {
		return s, err
	}

	// 4. Preferred Doctor (most visited)
	err = r.db.QueryRow(`
		SELECT a.doctor_id, d.full_name
		FROM appointments a
		JOIN doctors d ON d.id = a.doctor_id
		WHERE a.patient_id = $1 AND a.tenant_id = $2
		GROUP BY a.doctor_id, d.full_name
		ORDER BY COUNT(*) DESC LIMIT 1
	`, patientID, tenantID).Scan(&s.PreferredDoctorID, &s.PreferredDoctorName)
	if err != nil && err != sql.ErrNoRows {
		// Ignore error if no appointments yet
	}

	// 5. Invoices
	err = r.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'unpaid' OR status = 'partial') as unpaid
		FROM invoices
		WHERE patient_id = $1 AND tenant_id = $2
	`, patientID, tenantID).Scan(&s.TotalInvoices, &s.UnpaidInvoicesCount)
	if err != nil && err != sql.ErrNoRows {
		return s, err
	}

	// 6. Medical Records and Reports
	err = r.db.QueryRow(`SELECT COUNT(*) FROM medical_records WHERE patient_id = $1 AND tenant_id = $2`, patientID, tenantID).Scan(&s.MedicalRecordsCount)
	if err != nil && err != sql.ErrNoRows {
		return s, err
	}
	err = r.db.QueryRow(`SELECT COUNT(*) FROM attachments WHERE patient_id = $1 AND tenant_id = $2`, patientID, tenantID).Scan(&s.AttachmentsCount)
	if err != nil && err != sql.ErrNoRows {
		return s, err
	}

	return s, nil
}

func (r *postgresRepository) GetRecentAppointments(tenantID, patientID uuid.UUID, limit int) ([]RecentActivity, error) {
	rows, err := r.db.Query(`
		SELECT a.id, COALESCE(d.full_name, 'Unknown'), a.start_time, a.status
		FROM appointments a
		LEFT JOIN doctors d ON d.id = a.doctor_id
		WHERE a.patient_id = $1 AND a.tenant_id = $2
		ORDER BY a.start_time DESC LIMIT $3
	`, patientID, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := []RecentActivity{}
	for rows.Next() {
		var act RecentActivity
		act.Type = "appointment"
		if err := rows.Scan(&act.ID, &act.Subtitle, &act.Timestamp, &act.Status); err != nil {
			return nil, err
		}
		act.Title = "Appointment"
		activities = append(activities, act)
	}
	return activities, nil
}

func (r *postgresRepository) GetRecentMedicalRecords(tenantID, patientID uuid.UUID, limit int) ([]RecentActivity, error) {
	rows, err := r.db.Query(`
		SELECT id, diagnosis, created_at
		FROM medical_records
		WHERE patient_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC LIMIT $3
	`, patientID, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := []RecentActivity{}
	for rows.Next() {
		var act RecentActivity
		act.Type = "medical_record"
		if err := rows.Scan(&act.ID, &act.Title, &act.Timestamp); err != nil {
			return nil, err
		}
		act.Subtitle = "Medical Record"
		activities = append(activities, act)
	}
	return activities, nil
}

func (r *postgresRepository) GetRecentInvoices(tenantID, patientID uuid.UUID, limit int) ([]RecentActivity, error) {
	rows, err := r.db.Query(`
		SELECT id, total_amount, status, created_at
		FROM invoices
		WHERE patient_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC LIMIT $3
	`, patientID, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := []RecentActivity{}
	for rows.Next() {
		var act RecentActivity
		var amount float64
		act.Type = "invoice"
		if err := rows.Scan(&act.ID, &amount, &act.Status, &act.Timestamp); err != nil {
			return nil, err
		}
		act.Title = "Invoice"
		act.Subtitle = "Amount: " + act.Status // Simplified for now
		activities = append(activities, act)
	}
	return activities, nil
}

func (r *postgresRepository) GetRecentReports(tenantID, patientID uuid.UUID, limit int) ([]RecentActivity, error) {
	rows, err := r.db.Query(`
		SELECT id, filename, created_at
		FROM attachments
		WHERE patient_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC LIMIT $3
	`, patientID, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := []RecentActivity{}
	for rows.Next() {
		var act RecentActivity
		act.Type = "report"
		if err := rows.Scan(&act.ID, &act.Title, &act.Timestamp); err != nil {
			return nil, err
		}
		act.Subtitle = "Attachment"
		activities = append(activities, act)
	}
	return activities, nil
}
