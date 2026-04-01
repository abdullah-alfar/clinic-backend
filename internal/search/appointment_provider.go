package search

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type appointmentProvider struct {
	db *sql.DB
}

func NewAppointmentProvider(db *sql.DB) SearchProvider {
	return &appointmentProvider{db: db}
}

func (p *appointmentProvider) GetEntityType() EntityType {
	return EntityAppointment
}

func (p *appointmentProvider) GetEntityLabel() string {
	return "Appointments"
}

func (p *appointmentProvider) Search(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]SearchResultItem, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)

	q := `
		SELECT 
			a.id, 
			a.status, 
			a.start_time, 
			a.reason,
			p.first_name, 
			p.last_name,
			d.full_name
		FROM appointments a
		JOIN patients p ON a.patient_id = p.id
		JOIN doctors d ON a.doctor_id = d.id
		WHERE a.tenant_id = $1 
		  AND (
		      a.status ILIKE $2 OR 
		      a.reason ILIKE $2 OR 
		      p.first_name ILIKE $2 OR 
		      p.last_name ILIKE $2 OR 
		      d.full_name ILIKE $2 OR 
			  a.notes ILIKE $2
		  )
		ORDER BY a.start_time DESC
		LIMIT $3
	`

	rows, err := p.db.QueryContext(ctx, q, tenantID, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id string
		var status string
		var startTime time.Time
		var reason sql.NullString
		var pFName, pLName, dFullName string

		if err := rows.Scan(&id, &status, &startTime, &reason, &pFName, &pLName, &dFullName); err != nil {
			return nil, err
		}

		title := fmt.Sprintf("Appointment with %s %s", pFName, pLName)
		subtitle := fmt.Sprintf("%s • Dr. %s", startTime.Format("Jan 02, 2006 at 15:04"), dFullName)
		desc := "Status: " + status
		if reason.Valid && reason.String != "" {
			desc += " • Reason: " + reason.String
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       title,
			Subtitle:    subtitle,
			Description: desc,
			URL:         fmt.Sprintf("/appointments?id=%s", id), // or patient's tab
			Score:       0,
			Metadata: map[string]any{
				"status": status,
				"date":   startTime,
			},
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
