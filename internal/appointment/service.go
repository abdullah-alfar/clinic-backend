package appointment

import (
	"errors"
	"time"

	"clinic-backend/internal/audit"
	"clinic-backend/internal/queue"

	"github.com/google/uuid"
)

// Sentinel errors used throughout the appointment domain.
// Handlers map these to typed HTTP responses.
var (
	ErrDoubleBooking   = errors.New("doctor is already booked for this time slot")
	ErrDoctorInactive  = errors.New("doctor is not available during these hours")
	ErrInvalidTime     = errors.New("start time must be before end time")
	ErrPastAppointment = errors.New("cannot schedule appointment in the past")
	ErrNotFound        = errors.New("appointment not found")
	ErrInvalidStatus   = errors.New("invalid status transition")
	ErrNotMutable      = errors.New("appointment cannot be rescheduled in its current status")
)

// Appointment is the core domain model.
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

// AppointmentService orchestrates scheduling business rules.
// It does not contain persistence logic; that is delegated to AppointmentRepository.
type AppointmentService struct {
	repo  AppointmentRepository
	audit *audit.AuditService
	queue *queue.QueueClient
}

func NewAppointmentService(repo AppointmentRepository, audit *audit.AuditService, q *queue.QueueClient) *AppointmentService {
	return &AppointmentService{repo: repo, audit: audit, queue: q}
}

// isMutableStatus returns true when the appointment status allows rescheduling.
func isMutableStatus(status string) bool {
	return status == "scheduled" || status == "confirmed"
}

// CheckDoctorAvailability verifies that the requested time window falls within
// an active doctor_availability entry for the given tenant.
func (s *AppointmentService) CheckDoctorAvailability(tenantID, doctorID uuid.UUID, start, end time.Time) error {
	tz, _ := s.repo.GetTenantTimezone(tenantID)
	loc, _ := time.LoadLocation(tz)
	if loc == nil {
		loc = time.UTC
	}

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

// CheckConflict returns true when another non-canceled appointment overlaps the window.
// excludeID omits the current appointment itself (used during reschedule).
func (s *AppointmentService) CheckConflict(tenantID, doctorID uuid.UUID, start, end time.Time, excludeID *uuid.UUID) bool {
	count, _ := s.repo.CheckConflictCount(tenantID, doctorID, start, end, excludeID)
	return count > 0
}

// validateTimeWindow enforces ordering and past-time constraints shared by
// ScheduleAppointment, UpdateAppointmentTime, and RescheduleAppointment.
func (s *AppointmentService) validateTimeWindow(tenantID uuid.UUID, start, end time.Time) error {
	if start.After(end) || start.Equal(end) {
		return ErrInvalidTime
	}
	tz, _ := s.repo.GetTenantTimezone(tenantID)
	loc, _ := time.LoadLocation(tz)
	if loc == nil {
		loc = time.UTC
	}
	if start.Before(time.Now().In(loc)) {
		return ErrPastAppointment
	}
	return nil
}

// ScheduleAppointment creates a new appointment after validating all business rules.
func (s *AppointmentService) ScheduleAppointment(tenantID, patientID, doctorID uuid.UUID, start, end time.Time, createdBy uuid.UUID) (*Appointment, error) {
	if err := s.validateTimeWindow(tenantID, start, end); err != nil {
		return nil, err
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
	s.enqueueCreationEvents(tenantID, createdBy, appt.ID, patientID, start)

	return appt, nil
}

// RescheduleAppointment moves an existing appointment to a new time window.
// It reuses the same availability and conflict rules as ScheduleAppointment,
// ensuring a single authoritative code path for scheduling logic.
func (s *AppointmentService) RescheduleAppointment(tenantID, apptID uuid.UUID, start, end time.Time, actorID uuid.UUID) error {
	if err := s.validateTimeWindow(tenantID, start, end); err != nil {
		return err
	}

	doctorID, status, err := s.repo.GetAppointmentDoctorAndStatus(tenantID, apptID)
	if err != nil {
		return ErrNotFound
	}

	if !isMutableStatus(status) {
		return ErrNotMutable
	}

	if err := s.CheckDoctorAvailability(tenantID, doctorID, start, end); err != nil {
		return err
	}

	if s.CheckConflict(tenantID, doctorID, start, end, &apptID) {
		return ErrDoubleBooking
	}

	if err := s.repo.UpdateAppointmentTime(tenantID, apptID, start, end); err != nil {
		return err
	}

	s.audit.LogAction(tenantID, actorID, "RESCHEDULE_APPOINTMENT", "appointment", apptID, map[string]any{
		"start": start,
		"end":   end,
	})
	s.enqueueReminderUpdate(tenantID, apptID, start)

	return nil
}

// UpdateAppointmentTime handles ad-hoc time edits (not drag-and-drop reschedule).
// Kept for the existing PATCH /appointments/{id} route.
func (s *AppointmentService) UpdateAppointmentTime(tenantID, apptID uuid.UUID, start, end time.Time, actorID uuid.UUID) error {
	if err := s.validateTimeWindow(tenantID, start, end); err != nil {
		return err
	}

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

	if err := s.repo.UpdateAppointmentTime(tenantID, apptID, start, end); err != nil {
		return err
	}

	s.audit.LogAction(tenantID, actorID, "UPDATE_APPOINTMENT_TIME", "appointment", apptID, map[string]any{
		"start": start,
		"end":   end,
	})
	s.enqueueReminderUpdate(tenantID, apptID, start)

	return nil
}

// UpdateStatus transitions an appointment through its allowed status lifecycle.
func (s *AppointmentService) UpdateStatus(tenantID, apptID uuid.UUID, newStatus string, actorID uuid.UUID) error {
	_, currentStatus, err := s.repo.GetAppointmentDoctorAndStatus(tenantID, apptID)
	if err != nil {
		return err
	}

	if !isValidTransition(currentStatus, newStatus) {
		return ErrInvalidStatus
	}

	if err := s.repo.UpdateAppointmentStatus(tenantID, apptID, newStatus); err != nil {
		return err
	}

	s.audit.LogAction(tenantID, actorID, "UPDATE_APPOINTMENT_STATUS", "appointment", apptID, map[string]string{
		"old_status": currentStatus,
		"new_status": newStatus,
	})

	if s.queue != nil {
		s.queue.EnqueueNotification(queue.NotificationPayload{
			TenantID: tenantID.String(),
			UserID:   actorID.String(),
			Title:    "Appointment " + newStatus,
			Message:  "Appointment status was updated to " + newStatus,
			Type:     "appointment_status_changed",
		})
	}

	return nil
}

// GetCalendarAppointments retrieves enriched appointments for the calendar view.
// Date boundaries are normalized to clinic-local midnight to prevent timezone edge cases.
func (s *AppointmentService) GetCalendarAppointments(tenantID uuid.UUID, params CalendarQueryParams) ([]CalendarAppointment, string, error) {
	tz, _ := s.repo.GetTenantTimezone(tenantID)
	loc, _ := time.LoadLocation(tz)
	if loc == nil {
		loc = time.UTC
		tz = "UTC"
	}

	// Normalize boundaries to clinic-local midnight
	from := time.Date(params.DateFrom.Year(), params.DateFrom.Month(), params.DateFrom.Day(), 0, 0, 0, 0, loc)
	to := time.Date(params.DateTo.Year(), params.DateTo.Month(), params.DateTo.Day(), 23, 59, 59, 0, loc)

	var doctorIDs []uuid.UUID
	if params.DoctorID != nil {
		doctorIDs = []uuid.UUID{*params.DoctorID}
	}

	appointments, err := s.repo.GetCalendarAppointments(tenantID, doctorIDs, from, to)
	if err != nil {
		return nil, tz, err
	}

	return appointments, tz, nil
}

// isValidTransition enforces the appointment status state machine.
func isValidTransition(current, next string) bool {
	switch current {
	case "scheduled":
		return next == "confirmed" || next == "canceled"
	case "confirmed":
		return next == "completed" || next == "canceled"
	}
	return false
}

// enqueueCreationEvents fires booking confirmation and 24h reminder notifications.
func (s *AppointmentService) enqueueCreationEvents(tenantID, createdBy, apptID, patientID uuid.UUID, start time.Time) {
	if s.queue == nil {
		return
	}

	s.queue.EnqueueNotification(queue.NotificationPayload{
		TenantID: tenantID.String(),
		UserID:   createdBy.String(),
		Title:    "Appointment Booked",
		Message:  "A new appointment was scheduled successfully.",
		Type:     "appointment_created",
	})

	reminderTime := start.Add(-24 * time.Hour)
	if reminderTime.After(time.Now()) {
		s.queue.EnqueueReminder(queue.ReminderEmailPayload{
			TenantID:      tenantID.String(),
			AppointmentID: apptID.String(),
			PatientID:     patientID.String(),
		}, reminderTime)
	}
}

// enqueueReminderUpdate schedules a new 24h reminder after a time change.
func (s *AppointmentService) enqueueReminderUpdate(tenantID, apptID uuid.UUID, start time.Time) {
	if s.queue == nil {
		return
	}

	reminderTime := start.Add(-24 * time.Hour)
	if reminderTime.After(time.Now()) {
		s.queue.EnqueueReminder(queue.ReminderEmailPayload{
			TenantID:      tenantID.String(),
			AppointmentID: apptID.String(),
		}, reminderTime)
	}
}
