package appointment

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// DoctorAvailability holds a doctor's configured working window for a given day.
type DoctorAvailability struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	DoctorID  uuid.UUID
	DayOfWeek int
	StartTime string
	EndTime   string
	IsActive  bool
}

// CalendarAppointment is the domain model returned by the calendar query.
// It carries joined patient and doctor names so the service layer does not
// need to perform additional lookups.
type CalendarAppointment struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	PatientID   uuid.UUID
	PatientName string
	DoctorID    uuid.UUID
	DoctorName  string
	Status      string
	StartTime   time.Time
	EndTime     time.Time
	Reason      *string
}

// AppointmentRepository defines the data access contract for the appointment module.
// All implementations must satisfy this interface; no business logic lives here.
type AppointmentRepository interface {
	CheckDoctorAvailabilityCount(tenantID, doctorID uuid.UUID, dayOfWeek int, startTimeStr, endTimeStr string) (int, error)
	CheckConflictCount(tenantID, doctorID uuid.UUID, start, end time.Time, excludeID *uuid.UUID) (int, error)
	CreateAppointment(appt *Appointment) error
	GetAppointmentDoctorAndStatus(tenantID, apptID uuid.UUID) (uuid.UUID, string, error)
	UpdateAppointmentTime(tenantID, apptID uuid.UUID, start, end time.Time) error
	UpdateAppointmentStatus(tenantID, apptID uuid.UUID, status string) error
	GetDoctorAvailabilities(tenantID uuid.UUID, doctorIDs []uuid.UUID, dayOfWeek int) ([]DoctorAvailability, error)
	GetAppointmentsInRange(tenantID uuid.UUID, doctorIDs []uuid.UUID, start, end time.Time) ([]Appointment, error)
	GetCalendarAppointments(tenantID uuid.UUID, doctorIDs []uuid.UUID, start, end time.Time) ([]CalendarAppointment, error)
	GetTenantTimezone(tenantID uuid.UUID) (string, error)
}

type postgresAppointmentRepository struct {
	db *sql.DB
}

func NewPostgresAppointmentRepository(db *sql.DB) AppointmentRepository {
	return &postgresAppointmentRepository{db: db}
}

func (r *postgresAppointmentRepository) CheckDoctorAvailabilityCount(tenantID, doctorID uuid.UUID, dayOfWeek int, startTimeStr, endTimeStr string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT count(1) FROM doctor_availability
		WHERE tenant_id = $1 AND doctor_id = $2 AND day_of_week = $3 AND is_active = true
		AND start_time <= $4 AND end_time >= $5
	`, tenantID, doctorID, dayOfWeek, startTimeStr, endTimeStr).Scan(&count)
	return count, err
}

func (r *postgresAppointmentRepository) CheckConflictCount(tenantID, doctorID uuid.UUID, start, end time.Time, excludeID *uuid.UUID) (int, error) {
	query := `
		SELECT count(1) FROM appointments 
		WHERE tenant_id = $1 AND doctor_id = $2 
		AND status != 'canceled'
		AND (start_time < $3 AND end_time > $4)
	`
	args := []interface{}{tenantID, doctorID, end, start}

	if excludeID != nil {
		query += " AND id != $5"
		args = append(args, *excludeID)
	}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

func (r *postgresAppointmentRepository) CreateAppointment(appt *Appointment) error {
	_, err := r.db.Exec(`
		INSERT INTO appointments (id, tenant_id, patient_id, doctor_id, status, start_time, end_time, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, appt.ID, appt.TenantID, appt.PatientID, appt.DoctorID, appt.Status, appt.StartTime, appt.EndTime, appt.CreatedBy)
	return err
}

func (r *postgresAppointmentRepository) GetAppointmentDoctorAndStatus(tenantID, apptID uuid.UUID) (uuid.UUID, string, error) {
	var doctorID uuid.UUID
	var status string
	err := r.db.QueryRow(`SELECT doctor_id, status FROM appointments WHERE id = $1 AND tenant_id = $2`, apptID, tenantID).Scan(&doctorID, &status)
	return doctorID, status, err
}

func (r *postgresAppointmentRepository) UpdateAppointmentTime(tenantID, apptID uuid.UUID, start, end time.Time) error {
	_, err := r.db.Exec(`UPDATE appointments SET start_time = $1, end_time = $2, updated_at = NOW() WHERE id = $3 AND tenant_id = $4`, start, end, apptID, tenantID)
	return err
}

func (r *postgresAppointmentRepository) UpdateAppointmentStatus(tenantID, apptID uuid.UUID, status string) error {
	_, err := r.db.Exec(`UPDATE appointments SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`, status, apptID, tenantID)
	return err
}

func (r *postgresAppointmentRepository) GetDoctorAvailabilities(tenantID uuid.UUID, doctorIDs []uuid.UUID, dayOfWeek int) ([]DoctorAvailability, error) {
	query := `
		SELECT id, tenant_id, doctor_id, day_of_week, start_time::text, end_time::text, is_active
		FROM doctor_availability
		WHERE tenant_id = $1 AND day_of_week = $2 AND is_active = true
	`
	args := []interface{}{tenantID, dayOfWeek}

	if len(doctorIDs) > 0 {
		query += " AND doctor_id = ANY($3)"
		args = append(args, pq.Array(doctorIDs))
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DoctorAvailability
	for rows.Next() {
		var a DoctorAvailability
		if err := rows.Scan(&a.ID, &a.TenantID, &a.DoctorID, &a.DayOfWeek, &a.StartTime, &a.EndTime, &a.IsActive); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, nil
}

func (r *postgresAppointmentRepository) GetAppointmentsInRange(tenantID uuid.UUID, doctorIDs []uuid.UUID, start, end time.Time) ([]Appointment, error) {
	query := `
		SELECT id, tenant_id, patient_id, doctor_id, status, start_time, end_time, created_by
		FROM appointments
		WHERE tenant_id = $1
		AND status != 'canceled'
		AND start_time < $2 AND end_time > $3
	`
	args := []interface{}{tenantID, end, start}

	if len(doctorIDs) > 0 {
		query += " AND doctor_id = ANY($4)"
		args = append(args, pq.Array(doctorIDs))
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Appointment
	for rows.Next() {
		var a Appointment
		if err := rows.Scan(&a.ID, &a.TenantID, &a.PatientID, &a.DoctorID, &a.Status, &a.StartTime, &a.EndTime, &a.CreatedBy); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, nil
}

// GetCalendarAppointments returns enriched appointments (with patient and doctor names)
// for the given tenant, optional doctor filter, and time range. Uses a single JOIN query
// to avoid N+1 lookups in the service layer.
func (r *postgresAppointmentRepository) GetCalendarAppointments(tenantID uuid.UUID, doctorIDs []uuid.UUID, start, end time.Time) ([]CalendarAppointment, error) {
	query := `
		SELECT
			a.id,
			a.tenant_id,
			a.patient_id,
			COALESCE(p.first_name || ' ' || p.last_name, 'Unknown Patient') AS patient_name,
			a.doctor_id,
			COALESCE(d.full_name, 'Unknown Doctor') AS doctor_name,
			a.status,
			a.start_time,
			a.end_time,
			a.reason
		FROM appointments a
		LEFT JOIN patients p ON p.id = a.patient_id AND p.tenant_id = a.tenant_id
		LEFT JOIN doctors d ON d.id = a.doctor_id AND d.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1
		AND a.start_time < $2
		AND a.end_time > $3
		ORDER BY a.start_time ASC
	`
	args := []interface{}{tenantID, end, start}

	if len(doctorIDs) > 0 {
		query = `
			SELECT
				a.id,
				a.tenant_id,
				a.patient_id,
				COALESCE(p.first_name || ' ' || p.last_name, 'Unknown Patient') AS patient_name,
				a.doctor_id,
				COALESCE(d.full_name, 'Unknown Doctor') AS doctor_name,
				a.status,
				a.start_time,
				a.end_time,
				a.reason
			FROM appointments a
			LEFT JOIN patients p ON p.id = a.patient_id AND p.tenant_id = a.tenant_id
			LEFT JOIN doctors d ON d.id = a.doctor_id AND d.tenant_id = a.tenant_id
			WHERE a.tenant_id = $1
			AND a.start_time < $2
			AND a.end_time > $3
			AND a.doctor_id = ANY($4)
			ORDER BY a.start_time ASC
		`
		args = append(args, pq.Array(doctorIDs))
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CalendarAppointment
	for rows.Next() {
		var a CalendarAppointment
		if err := rows.Scan(
			&a.ID, &a.TenantID,
			&a.PatientID, &a.PatientName,
			&a.DoctorID, &a.DoctorName,
			&a.Status, &a.StartTime, &a.EndTime,
			&a.Reason,
		); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, nil
}

func (r *postgresAppointmentRepository) GetTenantTimezone(tenantID uuid.UUID) (string, error) {
	var tz string
	err := r.db.QueryRow(`SELECT timezone FROM tenants WHERE id = $1`, tenantID).Scan(&tz)
	if err != nil {
		return "UTC", nil
	}
	return tz, nil
}
