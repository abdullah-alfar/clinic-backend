package search

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type scheduleProvider struct {
	db *sql.DB
}

func NewScheduleProvider(db *sql.DB) SearchProvider {
	return &scheduleProvider{db: db}
}

func (p *scheduleProvider) GetEntityType() EntityType {
	return EntityDoctorSchedule
}

func (p *scheduleProvider) GetEntityLabel() string {
	return "Doctor Schedules"
}

func (p *scheduleProvider) Search(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]SearchResultItem, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)

	q := `
		SELECT 
			ds.id, 
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
		ORDER BY d.full_name ASC, ds.day_of_week ASC
		LIMIT $3
	`

	rows, err := p.db.QueryContext(ctx, q, tenantID, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

	var results []SearchResultItem
	for rows.Next() {
		var id, fullName string
		var dayOfWeek int
		var startTime, endTime string
		if err := rows.Scan(&id, &fullName, &dayOfWeek, &startTime, &endTime); err != nil {
			return nil, err
		}

		dayName := "Unknown"
		if dayOfWeek >= 0 && dayOfWeek < len(days) {
			dayName = days[dayOfWeek]
		}

		title := fmt.Sprintf("Schedule for Dr. %s", fullName)
		subtitle := fmt.Sprintf("%s: %s - %s", dayName, startTime, endTime)

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       title,
			Subtitle:    subtitle,
			Description: "Availability Shift",
			URL:         fmt.Sprintf("/doctors/%s/availability", id), // or specific doctor id
			Score:       0,
			Metadata: map[string]any{
				"day": dayName,
			},
		})
	}

	return results, nil
}
