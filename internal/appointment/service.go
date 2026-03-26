package appointment

import (
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
	repo  AppointmentRepository
	audit *audit.AuditService
	queue *queue.QueueClient
}

func NewAppointmentService(repo AppointmentRepository, audit *audit.AuditService, q *queue.QueueClient) *AppointmentService {
	return &AppointmentService{repo: repo, audit: audit, queue: q}
}

// CheckDoctorAvailability verifies working hours against doctor_availability table
func (s *AppointmentService) CheckDoctorAvailability(tenantID, doctorID uuid.UUID, start, end time.Time) error {
	tz, _ := s.repo.GetTenantTimezone(tenantID)
	loc, _ := time.LoadLocation(tz)
	if loc == nil { loc = time.UTC }

	startInLoc := start.In(loc)
	dayOfWeek := int(startInLoc.Weekday())
	startTimeStr := startInLoc.Format("15:04:05")
	endTimeStr := end.In(loc).Format("15:04:05")


	count, err := s.repo.CheckDoctorAvailabilityCount(tenantID, doctorID, dayOfWeek, startTimeStr, endTimeStr)

	if err != nil || count == 0 {
		return ErrDoctorInactive
	}
	return nil
}

func (s *AppointmentService) CheckConflict(tenantID, doctorID uuid.UUID, start, end time.Time, excludeID *uuid.UUID) bool {
	count, _ := s.repo.CheckConflictCount(tenantID, doctorID, start, end, excludeID)
	return count > 0
}

func (s *AppointmentService) ScheduleAppointment(tenantID, patientID, doctorID uuid.UUID, start, end time.Time, createdBy uuid.UUID) (*Appointment, error) {
	if start.After(end) || start.Equal(end) {
		return nil, ErrInvalidTime
	}
	tz, _ := s.repo.GetTenantTimezone(tenantID)
	loc, _ := time.LoadLocation(tz)
	if loc == nil { loc = time.UTC }

	if start.Before(time.Now().In(loc)) {
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

	if err := s.repo.CreateAppointment(appt); err != nil {
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
	doctorID, status, err := s.repo.GetAppointmentDoctorAndStatus(tenantID, apptID)
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

	err = s.repo.UpdateAppointmentTime(tenantID, apptID, start, end)
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
	_, currentStatus, err := s.repo.GetAppointmentDoctorAndStatus(tenantID, apptID)
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

	err = s.repo.UpdateAppointmentStatus(tenantID, apptID, newStatus)
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
