package availability

import "errors"

// Sentinel errors for the availability domain.
// Handlers map these to typed HTTP responses; never expose raw DB errors.
var (
	ErrNotFound          = errors.New("availability record not found")
	ErrOverlappingShift  = errors.New("schedule overlaps an existing shift for this day")
	ErrOverlappingBreak  = errors.New("break overlaps an existing break or falls outside the shift")
	ErrInvalidTimeRange  = errors.New("start_time must be before end_time")
	ErrInvalidDayOfWeek  = errors.New("day_of_week must be between 0 (Sunday) and 6 (Saturday)")
	ErrExceptionConflict = errors.New("a date exception already exists for this date")
	ErrDoctorNotInTenant = errors.New("doctor does not belong to this tenant")
)
