package availability

import (
	"time"

	"github.com/google/uuid"
)

// ─── Schedule DTOs ────────────────────────────────────────────────────────────

// CreateScheduleRequest is the validated body for POST /doctors/{id}/availability/schedules.
type CreateScheduleRequest struct {
	DayOfWeek int    `json:"day_of_week"` // 0–6
	StartTime string `json:"start_time"`  // "HH:MM"
	EndTime   string `json:"end_time"`    // "HH:MM"
}

// UpdateScheduleRequest is the body for PATCH /doctors/{id}/availability/schedules/{sid}.
// All fields are optional; only non-zero values should be applied.
type UpdateScheduleRequest struct {
	StartTime *string `json:"start_time,omitempty"`
	EndTime   *string `json:"end_time,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

// ScheduleDTO is the public representation of a Schedule sent to clients.
type ScheduleDTO struct {
	ID        string `json:"id"`
	DoctorID  string `json:"doctor_id"`
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	IsActive  bool   `json:"is_active"`
}

// ─── Break DTOs ───────────────────────────────────────────────────────────────

// CreateBreakRequest is the body for POST /doctors/{id}/availability/schedules/{sid}/breaks.
type CreateBreakRequest struct {
	StartTime string `json:"start_time"` // "HH:MM"
	EndTime   string `json:"end_time"`   // "HH:MM"
	Label     string `json:"label"`
}

// BreakDTO is the public representation of a Break.
type BreakDTO struct {
	ID         string `json:"id"`
	ScheduleID string `json:"schedule_id"`
	DayOfWeek  int    `json:"day_of_week"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Label      string `json:"label"`
}

// ─── Exception DTOs ───────────────────────────────────────────────────────────

// CreateExceptionRequest is the body for POST /doctors/{id}/availability/exceptions.
type CreateExceptionRequest struct {
	Date      string        `json:"date"`                 // "YYYY-MM-DD"
	Type      ExceptionType `json:"type"`                 // "day_off" | "override"
	StartTime *string       `json:"start_time,omitempty"` // required when type=="override"
	EndTime   *string       `json:"end_time,omitempty"`
	Reason    *string       `json:"reason,omitempty"`
}

// ExceptionDTO is the public representation of an Exception.
type ExceptionDTO struct {
	ID        string        `json:"id"`
	DoctorID  string        `json:"doctor_id"`
	Date      string        `json:"date"`
	Type      ExceptionType `json:"type"`
	StartTime *string       `json:"start_time"`
	EndTime   *string       `json:"end_time"`
	Reason    *string       `json:"reason"`
}

// ─── Slot Query DTOs ──────────────────────────────────────────────────────────

// SlotQueryParams carries validated parameters for the availability slot query.
type SlotQueryParams struct {
	DateFrom     time.Time
	DateTo       time.Time
	DoctorID     uuid.UUID
	SlotDuration time.Duration // defaults to 30 min when not specified
}

// DoctorSlotsDTO is the response envelope for GET /doctors/{id}/availability.
type DoctorSlotsDTO struct {
	DoctorID string     `json:"doctor_id"`
	Timezone string     `json:"timezone"`
	Slots    []SlotDTO  `json:"slots"`
}

// SlotDTO is the wire representation of a Slot.
type SlotDTO struct {
	StartTime string     `json:"start_time"`
	EndTime   string     `json:"end_time"`
	Status    SlotStatus `json:"status"`
}

// ─── Full Schedule View ───────────────────────────────────────────────────────

// DoctorAvailabilityDTO is the full availability view returned by
// GET /doctors/{id}/availability/schedule — combines schedules, breaks, exceptions.
type DoctorAvailabilityDTO struct {
	DoctorID   string         `json:"doctor_id"`
	Schedules  []ScheduleDTO  `json:"schedules"`
	Breaks     []BreakDTO     `json:"breaks"`
	Exceptions []ExceptionDTO `json:"exceptions"`
}

// ─── Mapping helpers ─────────────────────────────────────────────────────────

func toScheduleDTO(s Schedule) ScheduleDTO {
	return ScheduleDTO{
		ID:        s.ID.String(),
		DoctorID:  s.DoctorID.String(),
		DayOfWeek: s.DayOfWeek,
		StartTime: s.StartTime,
		EndTime:   s.EndTime,
		IsActive:  s.IsActive,
	}
}

func toBreakDTO(b Break) BreakDTO {
	return BreakDTO{
		ID:         b.ID.String(),
		ScheduleID: b.ScheduleID.String(),
		DayOfWeek:  b.DayOfWeek,
		StartTime:  b.StartTime,
		EndTime:    b.EndTime,
		Label:      b.Label,
	}
}

func toExceptionDTO(e Exception) ExceptionDTO {
	return ExceptionDTO{
		ID:        e.ID.String(),
		DoctorID:  e.DoctorID.String(),
		Date:      e.Date.Format("2006-01-02"),
		Type:      e.Type,
		StartTime: e.StartTime,
		EndTime:   e.EndTime,
		Reason:    e.Reason,
	}
}

func toSlotDTO(s Slot) SlotDTO {
	return SlotDTO{
		StartTime: s.StartTime.UTC().Format(time.RFC3339),
		EndTime:   s.EndTime.UTC().Format(time.RFC3339),
		Status:    s.Status,
	}
}
