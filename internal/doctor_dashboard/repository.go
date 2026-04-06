package doctor_dashboard

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	GetDoctorByUserID(ctx context.Context, tenantID, userID uuid.UUID) (*DoctorSummary, error)
	GetStats(ctx context.Context, tenantID, doctorID, userID uuid.UUID, todayStart, todayEnd time.Time) (*DashboardStats, error)
	GetTodayAppointments(ctx context.Context, tenantID, doctorID uuid.UUID, todayStart, todayEnd time.Time) ([]AppointmentSummary, error)
	GetUpcomingAppointments(ctx context.Context, tenantID, doctorID uuid.UUID, todayEnd time.Time, limit int) ([]AppointmentSummary, error)
	GetRecentPatients(ctx context.Context, tenantID, doctorID uuid.UUID, limit int) ([]RecentPatient, error)
	GetRecentMedicalActivity(ctx context.Context, tenantID, doctorID uuid.UUID, limit int) ([]MedicalActivity, error)
	GetTenantTimezone(ctx context.Context, tenantID uuid.UUID) (string, error)
}

type postgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetDoctorByUserID(ctx context.Context, tenantID, userID uuid.UUID) (*DoctorSummary, error) {
	var d DoctorSummary
	err := r.db.QueryRowContext(ctx, `
		SELECT id, full_name, coalesce(specialty, '')
		FROM doctors
		WHERE tenant_id = $1 AND user_id = $2
	`, tenantID, userID).Scan(&d.ID, &d.FullName, &d.Specialty)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *postgresRepository) GetStats(ctx context.Context, tenantID, doctorID, userID uuid.UUID, todayStart, todayEnd time.Time) (*DashboardStats, error) {
	var stats DashboardStats

	// Appointments Today
	err := r.db.QueryRowContext(ctx, `
		SELECT count(1) FROM appointments
		WHERE tenant_id = $1 AND doctor_id = $2
		AND start_time >= $3 AND start_time < $4
		AND status != 'canceled'
	`, tenantID, doctorID, todayStart, todayEnd).Scan(&stats.AppointmentsToday)
	if err != nil {
		return nil, err
	}

	// Upcoming Total
	err = r.db.QueryRowContext(ctx, `
		SELECT count(1) FROM appointments
		WHERE tenant_id = $1 AND doctor_id = $2
		AND start_time >= $3
		AND status = 'scheduled'
	`, tenantID, doctorID, todayEnd).Scan(&stats.UpcomingTotal)
	if err != nil {
		return nil, err
	}

	// Completed Today
	err = r.db.QueryRowContext(ctx, `
		SELECT count(1) FROM appointments
		WHERE tenant_id = $1 AND doctor_id = $2
		AND start_time >= $3 AND start_time < $4
		AND status = 'completed'
	`, tenantID, doctorID, todayStart, todayEnd).Scan(&stats.CompletedToday)
	if err != nil {
		return nil, err
	}

	// No Show Today
	err = r.db.QueryRowContext(ctx, `
		SELECT count(1) FROM appointments
		WHERE tenant_id = $1 AND doctor_id = $2
		AND start_time >= $3 AND start_time < $4
		AND status = 'no_show'
	`, tenantID, doctorID, todayStart, todayEnd).Scan(&stats.NoShowToday)
	if err != nil {
		return nil, err
	}

	// Pending Notes (Visits with empty diagnosis/notes for today's completed appts)
	err = r.db.QueryRowContext(ctx, `
		SELECT count(1) FROM appointments a
		LEFT JOIN visits v ON v.appointment_id = a.id
		WHERE a.tenant_id = $1 AND a.doctor_id = $2
		AND a.start_time >= $3 AND a.start_time < $4
		AND a.status = 'completed'
		AND (v.id IS NULL OR v.diagnosis = '' OR v.notes = '')
	`, tenantID, doctorID, todayStart, todayEnd).Scan(&stats.PendingNotes)
	if err != nil {
		return nil, err
	}

	// Unread Notifications
	err = r.db.QueryRowContext(ctx, `
		SELECT count(1) FROM notifications
		WHERE tenant_id = $1 AND user_id = $2
		AND read_at IS NULL
	`, tenantID, userID).Scan(&stats.UnreadNotifications)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *postgresRepository) GetTodayAppointments(ctx context.Context, tenantID, doctorID uuid.UUID, todayStart, todayEnd time.Time) ([]AppointmentSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.patient_id, p.first_name || ' ' || p.last_name, a.start_time, a.end_time, a.status, coalesce(a.reason, '')
		FROM appointments a
		JOIN patients p ON p.id = a.patient_id
		WHERE a.tenant_id = $1 AND a.doctor_id = $2
		AND a.start_time >= $3 AND a.start_time < $4
		AND a.status != 'canceled'
		ORDER BY a.start_time ASC
	`, tenantID, doctorID, todayStart, todayEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appts []AppointmentSummary
	for rows.Next() {
		var a AppointmentSummary
		if err := rows.Scan(&a.ID, &a.PatientID, &a.PatientName, &a.StartTime, &a.EndTime, &a.Status, &a.Reason); err != nil {
			return nil, err
		}
		appts = append(appts, a)
	}
	return appts, nil
}

func (r *postgresRepository) GetUpcomingAppointments(ctx context.Context, tenantID, doctorID uuid.UUID, todayEnd time.Time, limit int) ([]AppointmentSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.patient_id, p.first_name || ' ' || p.last_name, a.start_time, a.end_time, a.status, coalesce(a.reason, '')
		FROM appointments a
		JOIN patients p ON p.id = a.patient_id
		WHERE a.tenant_id = $1 AND a.doctor_id = $2
		AND a.start_time >= $3
		AND a.status = 'scheduled'
		ORDER BY a.start_time ASC
		LIMIT $4
	`, tenantID, doctorID, todayEnd, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appts []AppointmentSummary
	for rows.Next() {
		var a AppointmentSummary
		if err := rows.Scan(&a.ID, &a.PatientID, &a.PatientName, &a.StartTime, &a.EndTime, &a.Status, &a.Reason); err != nil {
			return nil, err
		}
		appts = append(appts, a)
	}
	return appts, nil
}

func (r *postgresRepository) GetRecentPatients(ctx context.Context, tenantID, doctorID uuid.UUID, limit int) ([]RecentPatient, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT ON (p.id) p.id, p.first_name || ' ' || p.last_name, v.created_at, coalesce(v.notes, '')
		FROM patients p
		JOIN visits v ON v.patient_id = p.id
		WHERE p.tenant_id = $1 AND v.doctor_id = $2
		ORDER BY p.id, v.created_at DESC
		LIMIT $3
	`, tenantID, doctorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patients = []RecentPatient{}
	for rows.Next() {
		var p RecentPatient
		if err := rows.Scan(&p.ID, &p.FullName, &p.LastVisit, &p.VisitNotes); err != nil {
			return nil, err
		}
		patients = append(patients, p)
	}
	return patients, nil
}

func (r *postgresRepository) GetRecentMedicalActivity(ctx context.Context, tenantID, doctorID uuid.UUID, limit int) ([]MedicalActivity, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT v.id, v.patient_id, p.first_name || ' ' || p.last_name, 'visit' as type, coalesce(v.diagnosis, 'No diagnosis'), v.created_at
		FROM visits v
		JOIN patients p ON p.id = v.patient_id
		WHERE v.tenant_id = $1 AND v.doctor_id = $2
		ORDER BY v.created_at DESC
		LIMIT $3
	`, tenantID, doctorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities = []MedicalActivity{}
	for rows.Next() {
		var m MedicalActivity
		if err := rows.Scan(&m.ID, &m.PatientID, &m.PatientName, &m.Type, &m.Description, &m.ActivityDate); err != nil {
			return nil, err
		}
		activities = append(activities, m)
	}
	return activities, nil
}

func (r *postgresRepository) GetTenantTimezone(ctx context.Context, tenantID uuid.UUID) (string, error) {
	var tz string
	err := r.db.QueryRowContext(ctx, "SELECT timezone FROM tenants WHERE id = $1", tenantID).Scan(&tz)
	if err != nil {
		return "UTC", nil
	}
	return tz, nil
}
