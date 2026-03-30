package availability

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// AvailabilityRepository defines the data-access contract for the availability domain.
// All methods are context-aware and tenant-scoped; no raw SQL escapes this boundary.
type AvailabilityRepository interface {
	// Schedule methods
	CreateSchedule(ctx context.Context, s *Schedule) error
	GetScheduleByID(ctx context.Context, tenantID, scheduleID uuid.UUID) (*Schedule, error)
	GetSchedulesByDoctor(ctx context.Context, tenantID, doctorID uuid.UUID) ([]Schedule, error)
	GetSchedulesByDoctorAndDay(ctx context.Context, tenantID, doctorID uuid.UUID, dayOfWeek int) ([]Schedule, error)
	UpdateSchedule(ctx context.Context, s *Schedule) error
	DeleteSchedule(ctx context.Context, tenantID, scheduleID uuid.UUID) error

	// Break methods
	CreateBreak(ctx context.Context, b *Break) error
	GetBreakByID(ctx context.Context, tenantID, breakID uuid.UUID) (*Break, error)
	GetBreaksBySchedule(ctx context.Context, tenantID, scheduleID uuid.UUID) ([]Break, error)
	GetBreaksByDoctorAndDay(ctx context.Context, tenantID, doctorID uuid.UUID, dayOfWeek int) ([]Break, error)
	DeleteBreak(ctx context.Context, tenantID, breakID uuid.UUID) error

	// Exception methods
	CreateException(ctx context.Context, e *Exception) error
	GetExceptionByID(ctx context.Context, tenantID, exceptionID uuid.UUID) (*Exception, error)
	GetExceptionsByDoctor(ctx context.Context, tenantID, doctorID uuid.UUID) ([]Exception, error)
	GetExceptionForDate(ctx context.Context, tenantID, doctorID uuid.UUID, date time.Time) (*Exception, error)
	DeleteException(ctx context.Context, tenantID, exceptionID uuid.UUID) error

	// Appointment overlap — delegated to the DB to keep slot-calculation pure
	GetBookedAppointmentSlots(ctx context.Context, tenantID, doctorID uuid.UUID, from, to time.Time) ([]bookedSlot, error)

	// Tenant timezone
	GetTenantTimezone(ctx context.Context, tenantID uuid.UUID) (string, error)

	// Doctor ownership guard — ensures doctor belongs to this tenant
	DoctorBelongsToTenant(ctx context.Context, tenantID, doctorID uuid.UUID) (bool, error)
}

// bookedSlot is a minimal projection of an existing appointment used purely for
// conflict detection inside slot generation. It is unexported deliberately; the
// domain outside this package should never depend on raw appointment intervals.
type bookedSlot struct {
	StartTime time.Time
	EndTime   time.Time
}

// postgresAvailabilityRepository is the Postgres-backed implementation.
type postgresAvailabilityRepository struct {
	db *sql.DB
}

// NewPostgresAvailabilityRepository constructs the production repository.
func NewPostgresAvailabilityRepository(db *sql.DB) AvailabilityRepository {
	return &postgresAvailabilityRepository{db: db}
}

// ─── Schedule ─────────────────────────────────────────────────────────────────

func (r *postgresAvailabilityRepository) CreateSchedule(ctx context.Context, s *Schedule) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO doctor_schedule
			(id, tenant_id, doctor_id, day_of_week, start_time, end_time, is_active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW(),NOW())
	`, s.ID, s.TenantID, s.DoctorID, s.DayOfWeek, s.StartTime, s.EndTime, s.IsActive)
	return err
}

func (r *postgresAvailabilityRepository) GetScheduleByID(ctx context.Context, tenantID, scheduleID uuid.UUID) (*Schedule, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, doctor_id, day_of_week,
		       start_time::text, end_time::text, is_active, created_at, updated_at
		FROM doctor_schedule
		WHERE id = $1 AND tenant_id = $2
	`, scheduleID, tenantID)
	return scanSchedule(row)
}

func (r *postgresAvailabilityRepository) GetSchedulesByDoctor(ctx context.Context, tenantID, doctorID uuid.UUID) ([]Schedule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, doctor_id, day_of_week,
		       start_time::text, end_time::text, is_active, created_at, updated_at
		FROM doctor_schedule
		WHERE tenant_id = $1 AND doctor_id = $2
		ORDER BY day_of_week, start_time
	`, tenantID, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSchedules(rows)
}

func (r *postgresAvailabilityRepository) GetSchedulesByDoctorAndDay(ctx context.Context, tenantID, doctorID uuid.UUID, dayOfWeek int) ([]Schedule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, doctor_id, day_of_week,
		       start_time::text, end_time::text, is_active, created_at, updated_at
		FROM doctor_schedule
		WHERE tenant_id = $1 AND doctor_id = $2 AND day_of_week = $3 AND is_active = true
		ORDER BY start_time
	`, tenantID, doctorID, dayOfWeek)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSchedules(rows)
}

func (r *postgresAvailabilityRepository) UpdateSchedule(ctx context.Context, s *Schedule) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE doctor_schedule
		SET start_time = $1, end_time = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4 AND tenant_id = $5
	`, s.StartTime, s.EndTime, s.IsActive, s.ID, s.TenantID)
	return err
}

func (r *postgresAvailabilityRepository) DeleteSchedule(ctx context.Context, tenantID, scheduleID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM doctor_schedule WHERE id = $1 AND tenant_id = $2`, scheduleID, tenantID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Breaks ───────────────────────────────────────────────────────────────────

func (r *postgresAvailabilityRepository) CreateBreak(ctx context.Context, b *Break) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO doctor_breaks
			(id, tenant_id, doctor_id, schedule_id, day_of_week, start_time, end_time, label, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
	`, b.ID, b.TenantID, b.DoctorID, b.ScheduleID, b.DayOfWeek, b.StartTime, b.EndTime, b.Label)
	return err
}

func (r *postgresAvailabilityRepository) GetBreakByID(ctx context.Context, tenantID, breakID uuid.UUID) (*Break, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, doctor_id, schedule_id, day_of_week,
		       start_time::text, end_time::text, label, created_at
		FROM doctor_breaks
		WHERE id = $1 AND tenant_id = $2
	`, breakID, tenantID)
	return scanBreak(row)
}

func (r *postgresAvailabilityRepository) GetBreaksBySchedule(ctx context.Context, tenantID, scheduleID uuid.UUID) ([]Break, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, doctor_id, schedule_id, day_of_week,
		       start_time::text, end_time::text, label, created_at
		FROM doctor_breaks
		WHERE tenant_id = $1 AND schedule_id = $2
		ORDER BY start_time
	`, tenantID, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBreaks(rows)
}

func (r *postgresAvailabilityRepository) GetBreaksByDoctorAndDay(ctx context.Context, tenantID, doctorID uuid.UUID, dayOfWeek int) ([]Break, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, doctor_id, schedule_id, day_of_week,
		       start_time::text, end_time::text, label, created_at
		FROM doctor_breaks
		WHERE tenant_id = $1 AND doctor_id = $2 AND day_of_week = $3
		ORDER BY start_time
	`, tenantID, doctorID, dayOfWeek)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBreaks(rows)
}

func (r *postgresAvailabilityRepository) DeleteBreak(ctx context.Context, tenantID, breakID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM doctor_breaks WHERE id = $1 AND tenant_id = $2`, breakID, tenantID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Exceptions ───────────────────────────────────────────────────────────────

func (r *postgresAvailabilityRepository) CreateException(ctx context.Context, e *Exception) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO doctor_exceptions
			(id, tenant_id, doctor_id, date, type, start_time, end_time, reason, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW())
	`, e.ID, e.TenantID, e.DoctorID, e.Date, string(e.Type), e.StartTime, e.EndTime, e.Reason)
	return err
}

func (r *postgresAvailabilityRepository) GetExceptionByID(ctx context.Context, tenantID, exceptionID uuid.UUID) (*Exception, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, doctor_id, date, type,
		       start_time::text, end_time::text, reason, created_at, updated_at
		FROM doctor_exceptions
		WHERE id = $1 AND tenant_id = $2
	`, exceptionID, tenantID)
	return scanException(row)
}

func (r *postgresAvailabilityRepository) GetExceptionsByDoctor(ctx context.Context, tenantID, doctorID uuid.UUID) ([]Exception, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, doctor_id, date, type,
		       start_time::text, end_time::text, reason, created_at, updated_at
		FROM doctor_exceptions
		WHERE tenant_id = $1 AND doctor_id = $2
		ORDER BY date DESC
	`, tenantID, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExceptions(rows)
}

func (r *postgresAvailabilityRepository) GetExceptionForDate(ctx context.Context, tenantID, doctorID uuid.UUID, date time.Time) (*Exception, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, doctor_id, date, type,
		       start_time::text, end_time::text, reason, created_at, updated_at
		FROM doctor_exceptions
		WHERE tenant_id = $1 AND doctor_id = $2 AND date = $3
	`, tenantID, doctorID, date.Format("2006-01-02"))
	e, err := scanException(row)
	if err == sql.ErrNoRows {
		return nil, nil // nil means "no exception for this date"
	}
	return e, err
}

func (r *postgresAvailabilityRepository) DeleteException(ctx context.Context, tenantID, exceptionID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM doctor_exceptions WHERE id = $1 AND tenant_id = $2`, exceptionID, tenantID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Appointment overlap ──────────────────────────────────────────────────────

func (r *postgresAvailabilityRepository) GetBookedAppointmentSlots(ctx context.Context, tenantID, doctorID uuid.UUID, from, to time.Time) ([]bookedSlot, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT start_time, end_time
		FROM appointments
		WHERE tenant_id = $1
		  AND doctor_id = $2
		  AND status != 'canceled'
		  AND start_time < $3
		  AND end_time   > $4
	`, tenantID, doctorID, to, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []bookedSlot
	for rows.Next() {
		var s bookedSlot
		if err := rows.Scan(&s.StartTime, &s.EndTime); err != nil {
			return nil, err
		}
		slots = append(slots, s)
	}
	return slots, nil
}

// ─── Tenant / Doctor helpers ──────────────────────────────────────────────────

func (r *postgresAvailabilityRepository) GetTenantTimezone(ctx context.Context, tenantID uuid.UUID) (string, error) {
	var tz string
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(timezone,'UTC') FROM tenants WHERE id = $1`, tenantID).Scan(&tz)
	if err != nil {
		return "UTC", nil
	}
	return tz, nil
}

func (r *postgresAvailabilityRepository) DoctorBelongsToTenant(ctx context.Context, tenantID, doctorID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM doctors WHERE id = $1 AND tenant_id = $2`,
		doctorID, tenantID).Scan(&count)
	return count > 0, err
}

// ─── Scan helpers ─────────────────────────────────────────────────────────────

func scanSchedule(row *sql.Row) (*Schedule, error) {
	var s Schedule
	err := row.Scan(
		&s.ID, &s.TenantID, &s.DoctorID,
		&s.DayOfWeek, &s.StartTime, &s.EndTime,
		&s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &s, err
}

func scanSchedules(rows *sql.Rows) ([]Schedule, error) {
	var list []Schedule
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.DoctorID,
			&s.DayOfWeek, &s.StartTime, &s.EndTime,
			&s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func scanBreak(row *sql.Row) (*Break, error) {
	var b Break
	err := row.Scan(
		&b.ID, &b.TenantID, &b.DoctorID, &b.ScheduleID,
		&b.DayOfWeek, &b.StartTime, &b.EndTime,
		&b.Label, &b.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &b, err
}

func scanBreaks(rows *sql.Rows) ([]Break, error) {
	var list []Break
	for rows.Next() {
		var b Break
		if err := rows.Scan(
			&b.ID, &b.TenantID, &b.DoctorID, &b.ScheduleID,
			&b.DayOfWeek, &b.StartTime, &b.EndTime,
			&b.Label, &b.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func scanException(row *sql.Row) (*Exception, error) {
	var e Exception
	var dateStr string
	var startTime, endTime sql.NullString
	err := row.Scan(
		&e.ID, &e.TenantID, &e.DoctorID,
		&dateStr, &e.Type,
		&startTime, &endTime,
		&e.Reason, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	parsed, _ := time.Parse("2006-01-02", dateStr)
	e.Date = parsed
	if startTime.Valid {
		e.StartTime = &startTime.String
	}
	if endTime.Valid {
		e.EndTime = &endTime.String
	}
	return &e, nil
}

func scanExceptions(rows *sql.Rows) ([]Exception, error) {
	var list []Exception
	for rows.Next() {
		var e Exception
		var dateStr string
		var startTime, endTime sql.NullString
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.DoctorID,
			&dateStr, &e.Type,
			&startTime, &endTime,
			&e.Reason, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		parsed, _ := time.Parse("2006-01-02", dateStr)
		e.Date = parsed
		if startTime.Valid {
			e.StartTime = &startTime.String
		}
		if endTime.Valid {
			e.EndTime = &endTime.String
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// ─── Overlap check helper (used by service through exported type) ─────────────

// CheckScheduleOverlap returns true when a proposed (dayOfWeek, start, end) window
// overlaps any existing active schedule for this doctor/tenant.
// Kept in the repo layer so the DB index on (tenant_id, doctor_id, day_of_week) is leveraged.
func (r *postgresAvailabilityRepository) checkScheduleOverlap(ctx context.Context, tenantID, doctorID uuid.UUID, dayOfWeek int, start, end string, excludeID *uuid.UUID) (bool, error) {
	query := `
		SELECT COUNT(1) FROM doctor_schedule
		WHERE tenant_id   = $1
		  AND doctor_id   = $2
		  AND day_of_week = $3
		  AND is_active   = true
		  AND start_time  < $4::time
		  AND end_time    > $5::time
	`
	args := []interface{}{tenantID, doctorID, dayOfWeek, end, start}

	if excludeID != nil {
		query += fmt.Sprintf(" AND id != $%d", len(args)+1)
		args = append(args, pq.Array([]uuid.UUID{*excludeID}))
		// rewrite to avoid array: use direct param
		query = `
			SELECT COUNT(1) FROM doctor_schedule
			WHERE tenant_id   = $1
			  AND doctor_id   = $2
			  AND day_of_week = $3
			  AND is_active   = true
			  AND start_time  < $4::time
			  AND end_time    > $5::time
			  AND id         != $6
		`
		args = []interface{}{tenantID, doctorID, dayOfWeek, end, start, *excludeID}
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count > 0, err
}
