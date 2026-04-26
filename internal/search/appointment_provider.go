package search

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type appointmentProvider struct{ db *sql.DB }

// NewAppointmentProvider creates a SearchProvider that searches appointments.
func NewAppointmentProvider(db *sql.DB) SearchProvider { return &appointmentProvider{db: db} }

func (p *appointmentProvider) Type() EntityType { return EntityAppointment }
func (p *appointmentProvider) Label() string    { return "Appointments" }

func (p *appointmentProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResultItem, error) {
	pattern := "%" + req.Query + "%"

	// Build dynamic WHERE clause for optional filters.
	args := []any{req.TenantID, pattern}
	extra := ""

	if req.Status != "" {
		args = append(args, req.Status)
		extra += fmt.Sprintf(" AND a.status = $%d", len(args))
	}
	if req.DateFrom != nil {
		args = append(args, *req.DateFrom)
		extra += fmt.Sprintf(" AND a.start_time >= $%d", len(args))
	}
	if req.DateTo != nil {
		args = append(args, *req.DateTo)
		extra += fmt.Sprintf(" AND a.start_time <= $%d", len(args))
	}
	if req.DoctorID != nil {
		args = append(args, *req.DoctorID)
		extra += fmt.Sprintf(" AND a.doctor_id = $%d", len(args))
	}
	if req.PatientID != nil {
		args = append(args, *req.PatientID)
		extra += fmt.Sprintf(" AND a.patient_id = $%d", len(args))
	}

	args = append(args, req.Limit)
	limitIdx := len(args)

	q := fmt.Sprintf(`
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
		JOIN doctors  d ON a.doctor_id  = d.id
		WHERE a.tenant_id = $1
		  AND (
		        a.status ILIKE $2 OR
		        a.reason ILIKE $2 OR
		        p.first_name ILIKE $2 OR
		        p.last_name  ILIKE $2 OR
		        (p.first_name || ' ' || p.last_name) ILIKE $2 OR
		        d.full_name ILIKE $2 OR
		        a.notes ILIKE $2
		      )
		%s
		ORDER BY a.start_time DESC
		LIMIT $%d
	`, extra, limitIdx)

	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("appointments: %w", err)
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id, status string
		var startTime time.Time
		var reason sql.NullString
		var pFName, pLName, dFullName string

		if err := rows.Scan(&id, &status, &startTime, &reason, &pFName, &pLName, &dFullName); err != nil {
			return nil, fmt.Errorf("appointments scan: %w", err)
		}

		desc := "Status: " + status
		if reason.Valid && reason.String != "" {
			desc += " • Reason: " + reason.String
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       "Appointment with " + pFName + " " + pLName,
			Subtitle:    startTime.Format("Jan 02, 2006 at 15:04") + " • Dr. " + dFullName,
			Description: desc,
			URL:         fmt.Sprintf("/appointments?id=%s", id),
			Metadata: map[string]any{
				"status":     status,
				"created_at": startTime,
			},
		})
	}
	return results, rows.Err()
}
