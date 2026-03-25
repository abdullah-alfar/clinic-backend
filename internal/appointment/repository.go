package appointment

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type DoctorAvailability struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	DoctorID  uuid.UUID
	DayOfWeek int
	StartTime string
	EndTime   string
	IsActive  bool
}

type AppointmentRepository interface {
	CheckDoctorAvailabilityCount(tenantID, doctorID uuid.UUID, dayOfWeek int, startTimeStr, endTimeStr string) (int, error)
	CheckConflictCount(tenantID, doctorID uuid.UUID, start, end time.Time, excludeID *uuid.UUID) (int, error)
	CreateAppointment(appt *Appointment) error
	GetAppointmentDoctorAndStatus(tenantID, apptID uuid.UUID) (uuid.UUID, string, error)
	UpdateAppointmentTime(tenantID, apptID uuid.UUID, start, end time.Time) error
	UpdateAppointmentStatus(tenantID, apptID uuid.UUID, status string) error
	GetDoctorAvailabilities(tenantID uuid.UUID, doctorIDs []uuid.UUID, dayOfWeek int) ([]DoctorAvailability, error)
	GetAppointmentsInRange(tenantID uuid.UUID, doctorIDs []uuid.UUID, start, end time.Time) ([]Appointment, error)
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
