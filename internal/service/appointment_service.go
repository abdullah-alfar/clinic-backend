package service

import (
	"errors"
	"time"

	"clinic-backend/internal/models"
	"github.com/google/uuid"
)

var (
	ErrDoubleBooking   = errors.New("doctor is already booked for this time slot")
	ErrInvalidTime     = errors.New("start time must be before end time")
	ErrPastAppointment = errors.New("cannot schedule appointment in the past")
	ErrNotFound        = errors.New("appointment not found")
	ErrInvalidStatus   = errors.New("invalid status transition")
)

type AppointmentService struct {
	// In a real app, this would be a database repository
	appointments map[uuid.UUID]*models.Appointment
}

func NewAppointmentService() *AppointmentService {
	return &AppointmentService{
		appointments: make(map[uuid.UUID]*models.Appointment),
	}
}

// CheckConflict returns true if the doctor has an overlapping appointment that is not canceled
func (s *AppointmentService) CheckConflict(doctorID uuid.UUID, start, end time.Time, excludeApptID *uuid.UUID) bool {
	for _, appt := range s.appointments {
		if appt.DoctorID == doctorID && appt.Status != models.StatusCanceled {
			if excludeApptID != nil && appt.ID == *excludeApptID {
				continue
			}
			// overlap logic
			if start.Before(appt.EndTime) && end.After(appt.StartTime) {
				return true
			}
		}
	}
	return false
}

func (s *AppointmentService) ScheduleAppointment(tenantID, patientID, doctorID uuid.UUID, start, end time.Time) (*models.Appointment, error) {
	if start.After(end) || start.Equal(end) {
		return nil, ErrInvalidTime
	}
	if start.Before(time.Now()) {
		return nil, ErrPastAppointment
	}

	if s.CheckConflict(doctorID, start, end, nil) {
		return nil, ErrDoubleBooking
	}

	appt := &models.Appointment{
		ID:        uuid.New(),
		TenantID:  tenantID,
		PatientID: patientID,
		DoctorID:  doctorID,
		StartTime: start,
		EndTime:   end,
		Status:    models.StatusScheduled,
		CreatedAt: time.Now(),
	}

	s.appointments[appt.ID] = appt
	return appt, nil
}

func (s *AppointmentService) RescheduleAppointment(id uuid.UUID, newStart, newEnd time.Time) (*models.Appointment, error) {
	appt, exists := s.appointments[id]
	if !exists {
		return nil, ErrNotFound
	}
	if appt.Status == models.StatusCanceled || appt.Status == models.StatusCompleted {
		return nil, ErrInvalidStatus
	}

	if newStart.After(newEnd) || newStart.Equal(newEnd) {
		return nil, ErrInvalidTime
	}
	if newStart.Before(time.Now()) {
		return nil, ErrPastAppointment
	}

	if s.CheckConflict(appt.DoctorID, newStart, newEnd, &id) {
		return nil, ErrDoubleBooking
	}

	appt.StartTime = newStart
	appt.EndTime = newEnd
	return appt, nil
}

func (s *AppointmentService) UpdateStatus(id uuid.UUID, status models.AppointmentStatus) (*models.Appointment, error) {
	appt, exists := s.appointments[id]
	if !exists {
		return nil, ErrNotFound
	}

	switch status {
	case models.StatusConfirmed, models.StatusCanceled, models.StatusCompleted:
		appt.Status = status
	default:
		return nil, ErrInvalidStatus
	}

	return appt, nil
}
