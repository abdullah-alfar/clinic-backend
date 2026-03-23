package service

import (
	"testing"
	"time"

	"clinic-backend/internal/models"
	"github.com/google/uuid"
)

func TestScheduleAppointment(t *testing.T) {
	svc := NewAppointmentService()
	tenantID := uuid.New()
	patientID := uuid.New()
	doctorID := uuid.New()

	now := time.Now()
	start := now.Add(24 * time.Hour)
	end := start.Add(1 * time.Hour)

	// Test 1: Successful schedule
	appt, err := svc.ScheduleAppointment(tenantID, patientID, doctorID, start, end)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if appt.Status != models.StatusScheduled {
		t.Errorf("expected status %v, got %v", models.StatusScheduled, appt.Status)
	}

	// Test 2: Double booking conflict
	overlapStart := start.Add(30 * time.Minute)
	overlapEnd := overlapStart.Add(1 * time.Hour)
	_, err = svc.ScheduleAppointment(tenantID, patientID, doctorID, overlapStart, overlapEnd)
	if err != ErrDoubleBooking {
		t.Errorf("expected ErrDoubleBooking, got %v", err)
	}

	// Test 3: Same time, different doctor
	doctor2ID := uuid.New()
	_, err = svc.ScheduleAppointment(tenantID, patientID, doctor2ID, start, end)
	if err != nil {
		t.Errorf("expected no error for different doctor, got %v", err)
	}
}

func TestRescheduleAppointment(t *testing.T) {
	svc := NewAppointmentService()
	tenantID := uuid.New()
	patientID := uuid.New()
	doctorID := uuid.New()

	now := time.Now()
	start := now.Add(24 * time.Hour)
	end := start.Add(1 * time.Hour)

	appt, _ := svc.ScheduleAppointment(tenantID, patientID, doctorID, start, end)

	// Block neighbor slot
	neighborStart := end
	neighborEnd := neighborStart.Add(1 * time.Hour)
	svc.ScheduleAppointment(tenantID, patientID, doctorID, neighborStart, neighborEnd)

	// Reschedule to overlap with neighbor
	rescheduleStart := neighborStart.Add(-30 * time.Minute)
	rescheduleEnd := rescheduleStart.Add(1 * time.Hour)
	_, err := svc.RescheduleAppointment(appt.ID, rescheduleStart, rescheduleEnd)
	if err != ErrDoubleBooking {
		t.Errorf("expected ErrDoubleBooking when rescheduling into conflict, got %v", err)
	}

	// Reschedule clear
	clearStart := now.Add(48 * time.Hour)
	clearEnd := clearStart.Add(1 * time.Hour)
	rescheduledAppt, err := svc.RescheduleAppointment(appt.ID, clearStart, clearEnd)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !rescheduledAppt.StartTime.Equal(clearStart) {
		t.Errorf("start time was not updated")
	}
}
