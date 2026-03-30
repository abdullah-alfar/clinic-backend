package availability

import (
	"time"

	"github.com/google/uuid"
)

// Schedule represents a doctor's recurring weekly working window on a given day.
// A doctor may have multiple shifts per day (e.g. morning + evening) — modelled
// as separate rows, allowing future multi-shift support without schema changes.
type Schedule struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	DoctorID  uuid.UUID `json:"doctor_id"`
	DayOfWeek int       `json:"day_of_week"` // 0=Sunday … 6=Saturday
	StartTime string    `json:"start_time"`  // "HH:MM:SS" stored as TIME in Postgres
	EndTime   string    `json:"end_time"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Break defines a recurring rest period within a doctor's shift on a given day.
// Breaks are stored separately so they can be added/removed without editing the
// parent schedule row, supporting future per-break audit trails.
type Break struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	DoctorID   uuid.UUID `json:"doctor_id"`
	ScheduleID uuid.UUID `json:"schedule_id"` // FK → doctor_schedule.id
	DayOfWeek  int       `json:"day_of_week"`
	StartTime  string    `json:"start_time"`
	EndTime    string    `json:"end_time"`
	Label      string    `json:"label"` // e.g. "Lunch", "Prayer"
	CreatedAt  time.Time `json:"created_at"`
}

// ExceptionType enumerates the kinds of date-specific overrides a doctor can have.
// Using a typed string keeps the domain expressive and avoids magic-number flags.
type ExceptionType string

const (
	ExceptionTypeDayOff   ExceptionType = "day_off"   // full day unavailable
	ExceptionTypeOverride ExceptionType = "override"   // custom hours on that date
)

// Exception is a date-specific override that supersedes the weekly schedule.
// When type is "day_off", start/end times are ignored during slot generation.
// When type is "override", the provided times replace the regular shift for that date.
// This design supports future exception types (e.g. "on_call") without migration.
type Exception struct {
	ID        uuid.UUID     `json:"id"`
	TenantID  uuid.UUID     `json:"tenant_id"`
	DoctorID  uuid.UUID     `json:"doctor_id"`
	Date      time.Time     `json:"date"`       // date-only; stored as DATE in Postgres
	Type      ExceptionType `json:"type"`
	StartTime *string       `json:"start_time"` // nil when type == "day_off"
	EndTime   *string       `json:"end_time"`
	Reason    *string       `json:"reason"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// Slot is a computed, time-bounded availability window returned to callers.
// It does not map to a DB table — it is produced by the AvailabilityService.
type Slot struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    SlotStatus `json:"status"`
}

// SlotStatus is an exhaustive enumeration of slot states, preventing stringly-typed
// comparisons across the codebase and making it safe to add new states later.
type SlotStatus string

const (
	SlotStatusAvailable   SlotStatus = "available"
	SlotStatusBooked      SlotStatus = "booked"
	SlotStatusBreak       SlotStatus = "break"
	SlotStatusUnavailable SlotStatus = "unavailable" // past slots
)
