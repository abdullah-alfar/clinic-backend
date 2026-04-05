package availability

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
)

// AvailabilityService encapsulates all availability domain logic.
// It depends only on the AvailabilityRepository interface — no DB details leak here.
type AvailabilityService struct {
	repo AvailabilityRepository
}

// NewAvailabilityService constructs the service with its required dependency.
func NewAvailabilityService(repo AvailabilityRepository) *AvailabilityService {
	return &AvailabilityService{repo: repo}
}

// ─── Timezone helper ──────────────────────────────────────────────────────────

func (s *AvailabilityService) tenantLocation(ctx context.Context, tenantID uuid.UUID) *time.Location {
	tz, _ := s.repo.GetTenantTimezone(ctx, tenantID)
	loc, err := time.LoadLocation(tz)
	if err != nil || loc == nil {
		return time.UTC
	}
	return loc
}

// ─── Schedule Operations ──────────────────────────────────────────────────────

// CreateSchedule adds a new weekly shift for a doctor.
// Validates day range, time ordering, and tenant-doctor ownership before persisting.
func (s *AvailabilityService) CreateSchedule(ctx context.Context, tenantID, doctorID uuid.UUID, req CreateScheduleRequest) (*ScheduleDTO, error) {
	if err := validateDayOfWeek(req.DayOfWeek); err != nil {
		return nil, err
	}
	if err := validateTimeRange(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if ok, err := s.repo.DoctorBelongsToTenant(ctx, tenantID, doctorID); err != nil || !ok {
		return nil, ErrDoctorNotInTenant
	}

	schedule := &Schedule{
		ID:        uuid.New(),
		TenantID:  tenantID,
		DoctorID:  doctorID,
		DayOfWeek: req.DayOfWeek,
		StartTime: normalizeTime(req.StartTime),
		EndTime:   normalizeTime(req.EndTime),
		IsActive:  true,
	}

	if err := s.repo.CreateSchedule(ctx, schedule); err != nil {
		return nil, err
	}

	dto := toScheduleDTO(*schedule)
	return &dto, nil
}

// GetDoctorSchedule retrieves all schedules (all days) for a doctor.
func (s *AvailabilityService) GetDoctorSchedule(ctx context.Context, tenantID, doctorID uuid.UUID) ([]ScheduleDTO, error) {
	if ok, err := s.repo.DoctorBelongsToTenant(ctx, tenantID, doctorID); err != nil || !ok {
		return nil, ErrDoctorNotInTenant
	}
	schedules, err := s.repo.GetSchedulesByDoctor(ctx, tenantID, doctorID)
	if err != nil {
		return nil, err
	}
	dtos := make([]ScheduleDTO, 0, len(schedules))
	for _, sc := range schedules {
		dtos = append(dtos, toScheduleDTO(sc))
	}
	return dtos, nil
}

// UpdateSchedule applies a partial update to an existing schedule entry.
func (s *AvailabilityService) UpdateSchedule(ctx context.Context, tenantID, scheduleID uuid.UUID, req UpdateScheduleRequest) (*ScheduleDTO, error) {
	existing, err := s.repo.GetScheduleByID(ctx, tenantID, scheduleID)
	if err != nil {
		return nil, err
	}

	// Merge only provided fields
	if req.StartTime != nil {
		existing.StartTime = normalizeTime(*req.StartTime)
	}
	if req.EndTime != nil {
		existing.EndTime = normalizeTime(*req.EndTime)
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := validateTimeRange(existing.StartTime, existing.EndTime); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateSchedule(ctx, existing); err != nil {
		return nil, err
	}

	dto := toScheduleDTO(*existing)
	return &dto, nil
}

// DeleteSchedule removes a schedule entry. Associated breaks are cascade-deleted by the DB.
func (s *AvailabilityService) DeleteSchedule(ctx context.Context, tenantID, scheduleID uuid.UUID) error {
	return s.repo.DeleteSchedule(ctx, tenantID, scheduleID)
}

// ─── Break Operations ─────────────────────────────────────────────────────────

// CreateBreak adds a recurring break within an existing schedule shift.
// The break must fall entirely inside the parent shift's time window.
func (s *AvailabilityService) CreateBreak(ctx context.Context, tenantID, doctorID, scheduleID uuid.UUID, req CreateBreakRequest) (*BreakDTO, error) {
	if err := validateTimeRange(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}

	// Ensure the parent schedule belongs to this tenant/doctor
	parent, err := s.repo.GetScheduleByID(ctx, tenantID, scheduleID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Break must lie within the parent shift
	if !timeWithin(req.StartTime, req.EndTime, parent.StartTime, parent.EndTime) {
		return nil, ErrOverlappingBreak
	}

	b := &Break{
		ID:         uuid.New(),
		TenantID:   tenantID,
		DoctorID:   doctorID,
		ScheduleID: scheduleID,
		DayOfWeek:  parent.DayOfWeek,
		StartTime:  normalizeTime(req.StartTime),
		EndTime:    normalizeTime(req.EndTime),
		Label:      req.Label,
	}

	if err := s.repo.CreateBreak(ctx, b); err != nil {
		return nil, err
	}

	dto := toBreakDTO(*b)
	return &dto, nil
}

// GetBreaksBySchedule returns all breaks for a given schedule slot.
func (s *AvailabilityService) GetBreaksBySchedule(ctx context.Context, tenantID, scheduleID uuid.UUID) ([]BreakDTO, error) {
	breaks, err := s.repo.GetBreaksBySchedule(ctx, tenantID, scheduleID)
	if err != nil {
		return nil, err
	}
	dtos := make([]BreakDTO, 0, len(breaks))
	for _, b := range breaks {
		dtos = append(dtos, toBreakDTO(b))
	}
	return dtos, nil
}

// DeleteBreak removes a break entry.
func (s *AvailabilityService) DeleteBreak(ctx context.Context, tenantID, breakID uuid.UUID) error {
	return s.repo.DeleteBreak(ctx, tenantID, breakID)
}

// ─── Exception Operations ─────────────────────────────────────────────────────

// CreateException registers a day-off or time-override for a specific date.
// Only one exception per doctor per date is permitted.
func (s *AvailabilityService) CreateException(ctx context.Context, tenantID, doctorID uuid.UUID, req CreateExceptionRequest) (*ExceptionDTO, error) {
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, ErrInvalidTimeRange
	}

	// Uniqueness guard — one exception per doctor per date
	existing, err := s.repo.GetExceptionForDate(ctx, tenantID, doctorID, date)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrExceptionConflict
	}

	if req.Type == ExceptionTypeOverride {
		if req.StartTime == nil || req.EndTime == nil {
			return nil, ErrInvalidTimeRange
		}
		st := normalizeTime(*req.StartTime)
		et := normalizeTime(*req.EndTime)
		if err := validateTimeRange(st, et); err != nil {
			return nil, err
		}
		req.StartTime = &st
		req.EndTime = &et
	}

	e := &Exception{
		ID:        uuid.New(),
		TenantID:  tenantID,
		DoctorID:  doctorID,
		Date:      date,
		Type:      req.Type,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Reason:    req.Reason,
	}

	if err := s.repo.CreateException(ctx, e); err != nil {
		return nil, err
	}

	dto := toExceptionDTO(*e)
	return &dto, nil
}

// GetExceptionsByDoctor returns all date exceptions for a doctor.
func (s *AvailabilityService) GetExceptionsByDoctor(ctx context.Context, tenantID, doctorID uuid.UUID) ([]ExceptionDTO, error) {
	if ok, err := s.repo.DoctorBelongsToTenant(ctx, tenantID, doctorID); err != nil || !ok {
		return nil, ErrDoctorNotInTenant
	}
	exceptions, err := s.repo.GetExceptionsByDoctor(ctx, tenantID, doctorID)
	if err != nil {
		return nil, err
	}
	dtos := make([]ExceptionDTO, 0, len(exceptions))
	for _, e := range exceptions {
		dtos = append(dtos, toExceptionDTO(e))
	}
	return dtos, nil
}

// DeleteException removes a date exception.
func (s *AvailabilityService) DeleteException(ctx context.Context, tenantID, exceptionID uuid.UUID) error {
	return s.repo.DeleteException(ctx, tenantID, exceptionID)
}

// ─── Full Schedule View ───────────────────────────────────────────────────────

// GetFullAvailability returns the combined schedules, breaks, and exceptions for a doctor.
// Useful for admin/settings UI that needs to display the complete availability configuration.
func (s *AvailabilityService) GetFullAvailability(ctx context.Context, tenantID, doctorID uuid.UUID) (*DoctorAvailabilityDTO, error) {
	if ok, err := s.repo.DoctorBelongsToTenant(ctx, tenantID, doctorID); err != nil || !ok {
		return nil, ErrDoctorNotInTenant
	}

	schedules, err := s.repo.GetSchedulesByDoctor(ctx, tenantID, doctorID)
	if err != nil {
		return nil, err
	}

	// Collect all breaks across every schedule
	var allBreaks []Break
	for _, sc := range schedules {
		br, err := s.repo.GetBreaksBySchedule(ctx, tenantID, sc.ID)
		if err != nil {
			return nil, err
		}
		allBreaks = append(allBreaks, br...)
	}

	exceptions, err := s.repo.GetExceptionsByDoctor(ctx, tenantID, doctorID)
	if err != nil {
		return nil, err
	}

	scheduleDTOs := make([]ScheduleDTO, 0, len(schedules))
	for _, sc := range schedules {
		scheduleDTOs = append(scheduleDTOs, toScheduleDTO(sc))
	}
	breakDTOs := make([]BreakDTO, 0, len(allBreaks))
	for _, b := range allBreaks {
		breakDTOs = append(breakDTOs, toBreakDTO(b))
	}
	exceptionDTOs := make([]ExceptionDTO, 0, len(exceptions))
	for _, e := range exceptions {
		exceptionDTOs = append(exceptionDTOs, toExceptionDTO(e))
	}

	return &DoctorAvailabilityDTO{
		DoctorID:   doctorID.String(),
		Schedules:  scheduleDTOs,
		Breaks:     breakDTOs,
		Exceptions: exceptionDTOs,
	}, nil
}

// ─── Slot Generation ──────────────────────────────────────────────────────────

// GetAvailableSlots computes the bookable time slots for a doctor over a date range.
//
// Algorithm (per day):
//  1. Check for a date exception — if day_off, skip day; if override, use override hours.
//  2. Load active schedule shifts for the day-of-week.
//  3. Load breaks for those shifts.
//  4. Fetch existing booked appointments for the day.
//  5. Walk each shift in slot-duration steps; mark each slot as:
//     - "booked"       — overlaps an existing appointment
//     - "break"        — falls within a configured break window
//     - "unavailable"  — slot start is in the past
//     - "available"    — none of the above
//
// The slot duration defaults to 30 minutes when not provided in params.
// The computation is pure (no side effects) and reusable by booking, calendar, and
// recurring-appointment modules without modification.
func (s *AvailabilityService) GetAvailableSlots(ctx context.Context, tenantID, doctorID uuid.UUID, params SlotQueryParams) (*DoctorSlotsDTO, error) {
	if ok, err := s.repo.DoctorBelongsToTenant(ctx, tenantID, doctorID); err != nil || !ok {
		return nil, ErrDoctorNotInTenant
	}

	loc := s.tenantLocation(ctx, tenantID)
	tz, _ := s.repo.GetTenantTimezone(ctx, tenantID)

	slotDuration := params.SlotDuration
	if slotDuration <= 0 {
		slotDuration = 30 * time.Minute
	}

	// Normalise date boundaries to clinic-local midnight
	from := time.Date(params.DateFrom.Year(), params.DateFrom.Month(), params.DateFrom.Day(), 0, 0, 0, 0, loc)
	to := time.Date(params.DateTo.Year(), params.DateTo.Month(), params.DateTo.Day(), 23, 59, 59, 0, loc)

	// Fetch all booked slots across the whole range in one query
	booked, err := s.repo.GetBookedAppointmentSlots(ctx, tenantID, doctorID, from, to)
	if err != nil {
		return nil, err
	}

	nowLocal := time.Now().In(loc)
	var slots []Slot

	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		date := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)

		// 1. Check for date exception
		exc, err := s.repo.GetExceptionForDate(ctx, tenantID, doctorID, date)
		if err != nil {
			return nil, err
		}
		if exc != nil && exc.Type == ExceptionTypeDayOff {
			continue // doctor is off this day
		}

		// 2. Build shift windows for this day
		type window struct{ start, end time.Time }
		var windows []window

		if exc != nil && exc.Type == ExceptionTypeOverride && exc.StartTime != nil && exc.EndTime != nil {
			// Use override hours instead of the weekly schedule
			ws := parseTimeToDayTime(*exc.StartTime, date, loc)
			we := parseTimeToDayTime(*exc.EndTime, date, loc)
			windows = append(windows, window{ws, we})
		} else {
			dayOfWeek := int(date.Weekday())
			schedules, err := s.repo.GetSchedulesByDoctorAndDay(ctx, tenantID, doctorID, dayOfWeek)
			if err != nil {
				return nil, err
			}
			for _, sc := range schedules {
				ws := parseTimeToDayTime(sc.StartTime, date, loc)
				we := parseTimeToDayTime(sc.EndTime, date, loc)
				windows = append(windows, window{ws, we})
			}
		}

		if len(windows) == 0 {
			continue // no schedule configured for this day
		}

		// 3. Load breaks for this day-of-week
		dayOfWeek := int(date.Weekday())
		breaks, err := s.repo.GetBreaksByDoctorAndDay(ctx, tenantID, doctorID, dayOfWeek)
		if err != nil {
			return nil, err
		}

		// 4. Generate slots per window
		for _, w := range windows {
			for cur := w.start; cur.Before(w.end); cur = cur.Add(slotDuration) {
				slotEnd := cur.Add(slotDuration)
				if slotEnd.After(w.end) {
					break // partial slot — skip
				}

				status := s.classifySlot(cur, slotEnd, nowLocal, booked, breaks, date, loc)
				slots = append(slots, Slot{StartTime: cur, EndTime: slotEnd, Status: status})
			}
		}
	}

	// Sort chronologically
	sort.Slice(slots, func(i, j int) bool {
		return slots[i].StartTime.Before(slots[j].StartTime)
	})

	dtos := make([]SlotDTO, 0, len(slots))
	for _, sl := range slots {
		dtos = append(dtos, toSlotDTO(sl))
	}

	return &DoctorSlotsDTO{
		DoctorID: doctorID.String(),
		Timezone: tz,
		Slots:    dtos,
	}, nil
}

// GetNextAvailableSlot finds the very next available slot within a 14-day rolling window.
func (s *AvailabilityService) GetNextAvailableSlot(ctx context.Context, tenantID, doctorID uuid.UUID) (*SlotDTO, error) {
	now := time.Now()
	// Look ahead up to 14 days
	maxDate := now.AddDate(0, 0, 14)

	slotsResp, err := s.GetAvailableSlots(ctx, tenantID, doctorID, SlotQueryParams{
		DateFrom: now,
		DateTo:   maxDate,
	})
	if err != nil {
		return nil, err
	}

	for _, slot := range slotsResp.Slots {
		if slot.Status == SlotStatusAvailable {
			return &slot, nil
		}
	}

	return nil, ErrNotFound
}

// classifySlot determines the status of a single slot based on past-time,
// break windows, and existing appointments. Order of precedence:
//  1. unavailable (past)
//  2. booked (appointment overlap)
//  3. break (break window overlap)
//  4. available
func (s *AvailabilityService) classifySlot(
	slotStart, slotEnd time.Time,
	now time.Time,
	booked []bookedSlot,
	breaks []Break,
	date time.Time,
	loc *time.Location,
) SlotStatus {
	if !slotStart.After(now) {
		return SlotStatusUnavailable
	}
	for _, b := range booked {
		if slotStart.Before(b.EndTime) && slotEnd.After(b.StartTime) {
			return SlotStatusBooked
		}
	}
	for _, br := range breaks {
		bStart := parseTimeToDayTime(br.StartTime, date, loc)
		bEnd := parseTimeToDayTime(br.EndTime, date, loc)
		if slotStart.Before(bEnd) && slotEnd.After(bStart) {
			return SlotStatusBreak
		}
	}
	return SlotStatusAvailable
}

// ─── Exported integration point for appointment scheduling ───────────────────

// IsDoctorAvailableAt checks whether a doctor has an active schedule covering the
// requested [start, end) window on the correct day, accounting for exceptions.
// This is the single authoritative availability check used by the appointment module.
func (s *AvailabilityService) IsDoctorAvailableAt(ctx context.Context, tenantID, doctorID uuid.UUID, start, end time.Time) error {
	loc := s.tenantLocation(ctx, tenantID)
	startLocal := start.In(loc)
	date := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), 0, 0, 0, 0, loc)

	// Check for date exception
	exc, err := s.repo.GetExceptionForDate(ctx, tenantID, doctorID, date)
	if err != nil {
		return err
	}
	if exc != nil && exc.Type == ExceptionTypeDayOff {
		return ErrNotFound // reuse sentinel; handler maps to "doctor unavailable"
	}
	if exc != nil && exc.Type == ExceptionTypeOverride {
		if exc.StartTime == nil || exc.EndTime == nil {
			return ErrNotFound
		}
		excStart := parseTimeToDayTime(*exc.StartTime, date, loc)
		excEnd := parseTimeToDayTime(*exc.EndTime, date, loc)
		if start.Before(excStart) || end.After(excEnd) {
			return ErrNotFound
		}
		return nil
	}

	// No exception — check weekly schedule
	dayOfWeek := int(startLocal.Weekday())
	schedules, err := s.repo.GetSchedulesByDoctorAndDay(ctx, tenantID, doctorID, dayOfWeek)
	if err != nil || len(schedules) == 0 {
		return ErrNotFound
	}

	startStr := startLocal.Format("15:04:05")
	endStr := end.In(loc).Format("15:04:05")

	for _, sc := range schedules {
		if sc.StartTime <= startStr && sc.EndTime >= endStr {
			return nil // found a covering shift
		}
	}
	return ErrNotFound
}

// ─── Pure helpers ─────────────────────────────────────────────────────────────

// parseTimeToDayTime combines a "HH:MM:SS" or "HH:MM" string with a calendar date
// in the given location to produce a fully-qualified time.Time. This avoids ambiguity
// when working across DST boundaries.
func parseTimeToDayTime(timeStr string, date time.Time, loc *time.Location) time.Time {
	t, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		t, _ = time.Parse("15:04", timeStr)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
}

// normalizeTime converts "HH:MM" → "HH:MM:00" for DB storage consistency.
func normalizeTime(t string) string {
	if len(t) == 5 {
		return t + ":00"
	}
	return t
}

// validateDayOfWeek enforces the 0–6 (Sun–Sat) range.
func validateDayOfWeek(d int) error {
	if d < 0 || d > 6 {
		return ErrInvalidDayOfWeek
	}
	return nil
}

// validateTimeRange ensures start strictly precedes end.
func validateTimeRange(start, end string) error {
	if start >= end {
		return ErrInvalidTimeRange
	}
	return nil
}

// timeWithin returns true when [inner start, inner end) is fully contained in [outer start, outer end).
func timeWithin(innerStart, innerEnd, outerStart, outerEnd string) bool {
	return innerStart >= outerStart && innerEnd <= outerEnd
}
