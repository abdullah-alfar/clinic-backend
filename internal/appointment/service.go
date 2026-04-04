package appointment

import (
	"context"
	"errors"
	"time"

	"clinic-backend/internal/audit"
	"clinic-backend/internal/notification"
	"clinic-backend/internal/queue"

	"github.com/google/uuid"
)

// Sentinel errors used throughout the appointment domain.
var (
	ErrDoubleBooking   = errors.New("doctor is already booked for this time slot")
	ErrDoctorInactive  = errors.New("doctor is not available during these hours")
	ErrInvalidTime     = errors.New("start time must be before end time")
	ErrPastAppointment = errors.New("cannot schedule appointment in the past")
	ErrNotFound        = errors.New("appointment not found")
	ErrInvalidStatus   = errors.New("invalid status transition")
	ErrNotMutable      = errors.New("appointment cannot be rescheduled in its current status")
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
	repo       AppointmentRepository
	audit      *audit.AuditService
	queue      *queue.QueueClient
	dispatcher notification.Dispatcher
}

func NewAppointmentService(repo AppointmentRepository, audit *audit.AuditService, q *queue.QueueClient, d notification.Dispatcher) *AppointmentService {
	return &AppointmentService{
		repo:       repo,
		audit:      audit,
		queue:      q,
		dispatcher: d,
	}
}

func (s *AppointmentService) dispatchEvent(tenantID, apptID, patientID, doctorID, actorID uuid.UUID, event string, start time.Time) {
	if s.dispatcher == nil {
		return
	}
	nd, err := s.repo.GetNotificationData(tenantID, patientID, doctorID)
	if err != nil {
		return
	}
	_ = s.dispatcher.Dispatch(context.Background(), notification.AppointmentEventPayload{
		TenantID:      tenantID,
		PatientID:     patientID,
		AppointmentID: apptID,
		ActorID:       actorID,
		Event:         event,
		PatientName:   nd.PatientName,
		PatientEmail:  nd.PatientEmail,
		PatientPhone:  nd.PatientPhone,
		DoctorName:    nd.DoctorName,
		ClinicName:    nd.ClinicName,
		StartTime:     start,
		Timezone:      nd.Timezone,
	})
}

func isMutableStatus(status string) bool {
	return status == "scheduled" || status == "confirmed"
}

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

func (s *AppointmentService) CheckConflict(tenantID, doctorID uuid.UUID, start, end time.Time, excludeID *uuid.UUID) bool {
	count, _ := s.repo.CheckConflictCount(tenantID, doctorID, start, end, excludeID)
	return count > 0
}

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
	s.dispatchEvent(tenantID, appt.ID, patientID, doctorID, createdBy, notification.EventAppointmentCreated, start)

	if s.queue != nil {
		s.queue.EnqueueNotification(queue.NotificationPayload{
			TenantID: tenantID.String(),
			UserID:   createdBy.String(),
			Title:    "Appointment Booked",
			Message:  "A new appointment was scheduled successfully.",
			Type:     "appointment_created",
		})
	}

	return appt, nil
}

func (s *AppointmentService) RescheduleAppointment(tenantID, apptID uuid.UUID, start, end time.Time, actorID uuid.UUID) error {
	if err := s.validateTimeWindow(tenantID, start, end); err != nil {
		return err
	}

	appt, err := s.repo.GetAppointmentByID(tenantID, apptID)
	if err != nil {
		return ErrNotFound
	}

	if !isMutableStatus(appt.Status) {
		return ErrNotMutable
	}

	if err := s.CheckDoctorAvailability(tenantID, appt.DoctorID, start, end); err != nil {
		return err
	}

	if s.CheckConflict(tenantID, appt.DoctorID, start, end, &apptID) {
		return ErrDoubleBooking
	}

	if err := s.repo.UpdateAppointmentTime(tenantID, apptID, start, end); err != nil {
		return err
	}

	s.audit.LogAction(tenantID, actorID, "RESCHEDULE_APPOINTMENT", "appointment", apptID, map[string]any{
		"start": start,
		"end":   end,
	})
	s.dispatchEvent(tenantID, apptID, appt.PatientID, appt.DoctorID, actorID, notification.EventAppointmentRescheduled, start)

	return nil
}

func (s *AppointmentService) UpdateAppointmentTime(tenantID, apptID uuid.UUID, start, end time.Time, actorID uuid.UUID) error {
	if err := s.validateTimeWindow(tenantID, start, end); err != nil {
		return err
	}

	appt, err := s.repo.GetAppointmentByID(tenantID, apptID)
	if err != nil {
		return err
	}

	if appt.Status == "canceled" || appt.Status == "completed" {
		return errors.New("cannot reschedule completed or canceled appointment")
	}

	if err := s.CheckDoctorAvailability(tenantID, appt.DoctorID, start, end); err != nil {
		return err
	}

	if s.CheckConflict(tenantID, appt.DoctorID, start, end, &apptID) {
		return ErrDoubleBooking
	}

	if err := s.repo.UpdateAppointmentTime(tenantID, apptID, start, end); err != nil {
		return err
	}

	s.audit.LogAction(tenantID, actorID, "UPDATE_APPOINTMENT_TIME", "appointment", apptID, map[string]any{
		"start": start,
		"end":   end,
	})
	s.dispatchEvent(tenantID, apptID, appt.PatientID, appt.DoctorID, actorID, notification.EventAppointmentRescheduled, start)

	return nil
}

func (s *AppointmentService) UpdateStatus(tenantID, apptID uuid.UUID, newStatus string, actorID uuid.UUID) error {
	appt, err := s.repo.GetAppointmentByID(tenantID, apptID)
	if err != nil {
		return err
	}

	if !isValidTransition(appt.Status, newStatus) {
		return ErrInvalidStatus
	}

	if err := s.repo.UpdateAppointmentStatus(tenantID, apptID, newStatus); err != nil {
		return err
	}

	s.audit.LogAction(tenantID, actorID, "UPDATE_APPOINTMENT_STATUS", "appointment", apptID, map[string]string{
		"old_status": appt.Status,
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

	var event string
	switch newStatus {
	case "confirmed":
		event = notification.EventAppointmentConfirmed
	case "canceled":
		event = notification.EventAppointmentCanceled
	}

	if event != "" {
		s.dispatchEvent(tenantID, apptID, appt.PatientID, appt.DoctorID, actorID, event, appt.StartTime)
	}

	return nil
}

func (s *AppointmentService) GetCalendarAppointments(tenantID uuid.UUID, params CalendarQueryParams) ([]CalendarAppointment, string, error) {
	tz, _ := s.repo.GetTenantTimezone(tenantID)
	loc, _ := time.LoadLocation(tz)
	if loc == nil {
		loc = time.UTC
		tz = "UTC"
	}

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

func isValidTransition(current, next string) bool {
	switch current {
	case "scheduled":
		return next == "confirmed" || next == "canceled"
	case "confirmed":
		return next == "completed" || next == "canceled"
	}
	return false
}
