package recurrence

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type RecurrenceRepository interface {
	CreateRule(ctx context.Context, r *RecurrenceRule) error
	GetRuleByID(ctx context.Context, tenantID, id uuid.UUID) (*RecurrenceRule, error)
	GetRulesByPatient(ctx context.Context, tenantID, patientID uuid.UUID) ([]RecurrenceRule, error)
	UpdateRuleStatus(ctx context.Context, tenantID, id uuid.UUID, status RecurrenceStatus) error
}

type postgresRecurrenceRepository struct {
	db *sql.DB
}

func NewPostgresRecurrenceRepository(db *sql.DB) RecurrenceRepository {
	return &postgresRecurrenceRepository{db: db}
}

func (r *postgresRecurrenceRepository) CreateRule(ctx context.Context, rule *RecurrenceRule) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO recurrence_rules (
			id, tenant_id, patient_id, doctor_id, frequency, interval, 
			day_of_week, day_of_month, start_time, end_time, start_date, end_date, 
			reason, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
	`, rule.ID, rule.TenantID, rule.PatientID, rule.DoctorID, rule.Frequency, rule.Interval,
		rule.DayOfWeek, rule.DayOfMonth, rule.StartTime, rule.EndTime, rule.StartDate, rule.EndDate,
		rule.Reason, rule.Status)
	return err
}

func (r *postgresRecurrenceRepository) GetRuleByID(ctx context.Context, tenantID, id uuid.UUID) (*RecurrenceRule, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, patient_id, doctor_id, frequency, interval, 
		       day_of_week, day_of_month, start_time::text, end_time::text, start_date, end_date, 
		       reason, status, created_at, updated_at
		FROM recurrence_rules
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID)

	var rule RecurrenceRule
	err := row.Scan(
		&rule.ID, &rule.TenantID, &rule.PatientID, &rule.DoctorID, &rule.Frequency, &rule.Interval,
		&rule.DayOfWeek, &rule.DayOfMonth, &rule.StartTime, &rule.EndTime, &rule.StartDate, &rule.EndDate,
		&rule.Reason, &rule.Status, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *postgresRecurrenceRepository) GetRulesByPatient(ctx context.Context, tenantID, patientID uuid.UUID) ([]RecurrenceRule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, patient_id, doctor_id, frequency, interval, 
		       day_of_week, day_of_month, start_time::text, end_time::text, start_date, end_date, 
		       reason, status, created_at, updated_at
		FROM recurrence_rules
		WHERE tenant_id = $1 AND patient_id = $2
		ORDER BY created_at DESC
	`, tenantID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []RecurrenceRule
	for rows.Next() {
		var rule RecurrenceRule
		err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.PatientID, &rule.DoctorID, &rule.Frequency, &rule.Interval,
			&rule.DayOfWeek, &rule.DayOfMonth, &rule.StartTime, &rule.EndTime, &rule.StartDate, &rule.EndDate,
			&rule.Reason, &rule.Status, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *postgresRecurrenceRepository) UpdateRuleStatus(ctx context.Context, tenantID, id uuid.UUID, status RecurrenceStatus) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE recurrence_rules SET status = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`, status, id, tenantID)
	return err
}
