package appointment

import (
	"database/sql"
	"errors"
	"time"

	"clinic-backend/internal/audit"
	"clinic-backend/internal/queue"
	"github.com/google/uuid"
)

var (
	ErrDoubleBooking   = errors.New("doctor is already booked for this time slot")
	ErrDoctorInactive  = errors.New("doctor is not available during these hours")
	ErrInvalidTime     = errors.New("start time must be before end time")
	ErrPastAppointment = errors.New("cannot schedule appointment in the past")
	ErrNotFound        = errors.New("appointment not found")
	ErrInvalidStatus   = errors.New("invalid status transition")
)

type Appointment struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	PatientID uuid.UUID  `json:"patient_id"`
	DoctorID  uuid.UUID  `json:"doctor_id"`
	Status    string     `json:"status"`
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time"`
	Reason    *string    `json:"reason"`
	CreatedBy *uuid.UUID `json:"created_by"`
}

type AppointmentService struct {
	db    *sql.DB
	audit *audit.AuditService
	queue *queue.QueueClient
}

func NewAppointmentService(db *sql.DB, audit *audit.AuditService, q *queue.QueueClient) *AppointmentService {
	return &AppointmentService{db: db, audit: audit, queue: q}
}

// CheckDoctorAvailability verifies working hours against doctor_availability table
func (s *AppointmentService) CheckDoctorAvailability(tenantID, doctorID uuid.UUID, start, end time.Time) error {
	var count int
	dayOfWeek := int(start.Weekday())
	startTimeStr := start.Format("15:04:05")
	endTimeStr := end.Format("15:04:05")

	err := s.db.QueryRow(`
		SELECT count(1) FROM doctor_availability
		WHERE tenant_id = $1 AND doctor_id = $2 AND day_of_week = $3 AND is_active = true
		AND start_time <= $4 AND end_time >= $5
	`, tenantID, doctorID, dayOfWeek, startTimeStr, endTimeStr).Scan(&count)

	if err != nil || count == 0 {
		return ErrDoctorInactive
	}
	return nil
}

func (s *AppointmentService) CheckConflict(tenantID, doctorID uuid.UUID, start, end time.Time, excludeID *uuid.UUID) bool {
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
	s.db.QueryRow(query, args...).Scan(&count)
	return count > 0
}

func (s *AppointmentService) ScheduleAppointment(tenantID, patientID, doctorID uuid.UUID, start, end time.Time, createdBy uuid.UUID) (*Appointment, error) {
	if start.After(end) || start.Equal(end) {
		return nil, ErrInvalidTime
	}
	if start.Before(time.Now()) {
		return nil, ErrPastAppointment
	}

	if err := s.CheckDoctorAvailability(tenantID, doctorID, start, end); err != nil {
		return nil, err
	}

	if s.CheckConflict(tenantID, doctorID, start, end, nil) {
		return nil, ErrDoubleBooking
	}

	appt := &Appointment{
		ID:        uuid.New(),
		TenantID:  tenantID,
		PatientID: patientID,
		DoctorID:  doctorID,
		Status:    "scheduled",
		StartTime: start,
		EndTime:   end,
		CreatedBy: &createdBy,
	}

	_, err := s.db.Exec(`
		INSERT INTO appointments (id, tenant_id, patient_id, doctor_id, status, start_time, end_time, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, appt.ID, appt.TenantID, appt.PatientID, appt.DoctorID, appt.Status, appt.StartTime, appt.EndTime, appt.CreatedBy)

	if err != nil {
		return nil, err
	}

	s.audit.LogAction(tenantID, createdBy, "CREATE_APPOINTMENT", "appointment", appt.ID, appt)

	// Phase 4: Enqueue Events silently
	if s.queue != nil {
		s.queue.EnqueueNotification(queue.NotificationPayload{
			TenantID: tenantID.String(),
			UserID:   createdBy.String(),
			Title:    "Appointment Booked",
			Message:  "A new appointment was scheduled successfully.",
			Type:     "appointment_created",
		})

		// Schedule reminder 24h before
		reminderTime := start.Add(-24 * time.Hour)
		if reminderTime.After(time.Now()) {
			s.queue.EnqueueReminder(queue.ReminderEmailPayload{
				TenantID:      tenantID.String(),
				AppointmentID: appt.ID.String(),
				PatientID:     patientID.String(),
			}, reminderTime)
		}
	}

	return appt, nil
}

func (s *AppointmentService) UpdateAppointmentTime(tenantID, apptID uuid.UUID, start, end time.Time, actorID uuid.UUID) error {
	if start.After(end) || start.Equal(end) {
		return ErrInvalidTime
	}
	if start.Before(time.Now()) {
		return ErrPastAppointment
	}

	// Fetch current appt
	var doctorID uuid.UUID
	var status string
	err := s.db.QueryRow(`SELECT doctor_id, status FROM appointments WHERE id = $1 AND tenant_id = $2`, apptID, tenantID).Scan(&doctorID, &status)
	if err != nil {
		return err
	}

	if status == "canceled" || status == "completed" {
		return errors.New("cannot reschedule completed or canceled appointment")
	}

	if err := s.CheckDoctorAvailability(tenantID, doctorID, start, end); err != nil {
		return err
	}

	if s.CheckConflict(tenantID, doctorID, start, end, &apptID) {
		return ErrDoubleBooking
	}

	_, err = s.db.Exec(`UPDATE appointments SET start_time = $1, end_time = $2, updated_at = NOW() WHERE id = $3 AND tenant_id = $4`, start, end, apptID, tenantID)
	if err == nil {
		s.audit.LogAction(tenantID, actorID, "UPDATE_APPOINTMENT_TIME", "appointment", apptID, map[string]any{"start": start, "end": end})
		
		if s.queue != nil {
			reminderTime := start.Add(-24 * time.Hour)
			if reminderTime.After(time.Now()) {
				s.queue.EnqueueReminder(queue.ReminderEmailPayload{
					TenantID:      tenantID.String(),
					AppointmentID: apptID.String(),
				}, reminderTime)
			}
		}
	}
	return err
}

func (s *AppointmentService) UpdateStatus(tenantID, apptID uuid.UUID, newStatus string, actorID uuid.UUID) error {
	var currentStatus string
	err := s.db.QueryRow(`SELECT status FROM appointments WHERE id = $1 AND tenant_id = $2`, apptID, tenantID).Scan(&currentStatus)
	if err != nil {
		return err
	}

	// Validate transitions
	valid := false
	switch currentStatus {
	case "scheduled":
		valid = newStatus == "confirmed" || newStatus == "canceled"
	case "confirmed":
		valid = newStatus == "completed" || newStatus == "canceled"
	}

	if !valid {
		return ErrInvalidStatus
	}

	_, err = s.db.Exec(`UPDATE appointments SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`, newStatus, apptID, tenantID)
	if err == nil {
		s.audit.LogAction(tenantID, actorID, "UPDATE_APPOINTMENT_STATUS", "appointment", apptID, map[string]string{"old_status": currentStatus, "new_status": newStatus})
		
		if s.queue != nil {
			s.queue.EnqueueNotification(queue.NotificationPayload{
				TenantID: tenantID.String(),
				UserID:   actorID.String(),
				Title:    "Appointment " + newStatus,
				Message:  "Appointment status was updated to " + newStatus,
				Type:     "appointment_status_changed",
			})
		}
	}
	return err
}
