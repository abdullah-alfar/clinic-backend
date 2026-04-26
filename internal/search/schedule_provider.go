package search

import (
	"context"
	"database/sql"
	"fmt"
)

type scheduleProvider struct{ db *sql.DB }

// NewScheduleProvider creates a SearchProvider that searches doctor availability schedules.
func NewScheduleProvider(db *sql.DB) SearchProvider { return &scheduleProvider{db: db} }

func (p *scheduleProvider) Type() EntityType { return EntityDoctorSchedule }
func (p *scheduleProvider) Label() string    { return "Doctor Schedules" }

var dayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

func (p *scheduleProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResultItem, error) {
	pattern := "%" + req.Query + "%"

	args := []any{req.TenantID, pattern}
	extra := ""

	if req.DoctorID != nil {
		args = append(args, *req.DoctorID)
		extra += fmt.Sprintf(" AND ds.doctor_id = $%d", len(args))
	}

	args = append(args, req.Limit)
	limitIdx := len(args)

	q := fmt.Sprintf(`
		SELECT
			ds.id,
			d.id,
			d.full_name,
			ds.day_of_week,
			ds.start_time,
			ds.end_time
		FROM doctor_schedule ds
		JOIN doctors d ON ds.doctor_id = d.id
		WHERE ds.tenant_id = $1
		  AND (
		        d.full_name ILIKE $2 OR
		        d.specialty ILIKE $2
		      )
		%s
		ORDER BY d.full_name ASC, ds.day_of_week ASC
		LIMIT $%d
	`, extra, limitIdx)

	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("schedules: %w", err)
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var scheduleID, doctorID, fullName string
		var dayOfWeek int
		var startTime, endTime string
		if err := rows.Scan(&scheduleID, &doctorID, &fullName, &dayOfWeek, &startTime, &endTime); err != nil {
			return nil, fmt.Errorf("schedules scan: %w", err)
		}

		day := "Unknown"
		if dayOfWeek >= 0 && dayOfWeek < len(dayNames) {
			day = dayNames[dayOfWeek]
		}

		results = append(results, SearchResultItem{
			ID:          scheduleID,
			Title:       "Schedule for Dr. " + fullName,
			Subtitle:    fmt.Sprintf("%s: %s - %s", day, startTime, endTime),
			Description: "Availability Shift",
			URL:         fmt.Sprintf("/doctors/%s/availability", doctorID),
			Metadata:    map[string]any{"day": day},
		})
	}
	return results, rows.Err()
}
