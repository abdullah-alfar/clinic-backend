package search

import (
	"context"
	"database/sql"
	"fmt"
)

type patientProvider struct{ db *sql.DB }

// NewPatientProvider creates a SearchProvider that searches the patients table.
func NewPatientProvider(db *sql.DB) SearchProvider { return &patientProvider{db: db} }

func (p *patientProvider) Type() EntityType { return EntityPatient }
func (p *patientProvider) Label() string    { return "Patients" }

func (p *patientProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResultItem, error) {
	pattern := "%" + req.Query + "%"

	q := `
		SELECT id, first_name, last_name, phone, email
		FROM patients
		WHERE tenant_id = $1
		  AND (
		        first_name ILIKE $2 OR
		        last_name  ILIKE $2 OR
		        (first_name || ' ' || last_name) ILIKE $2 OR
		        phone ILIKE $2 OR
		        email ILIKE $2
		      )
		ORDER BY last_name ASC, first_name ASC
		LIMIT $3
	`

	rows, err := p.db.QueryContext(ctx, q, req.TenantID, pattern, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("patients: %w", err)
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var id, fName, lName string
		var phone, email *string
		if err := rows.Scan(&id, &fName, &lName, &phone, &email); err != nil {
			return nil, fmt.Errorf("patients scan: %w", err)
		}

		subtitle := ""
		if phone != nil && *phone != "" {
			subtitle = *phone
		}
		if email != nil && *email != "" {
			if subtitle != "" {
				subtitle += " • "
			}
			subtitle += *email
		}

		results = append(results, SearchResultItem{
			ID:          id,
			Title:       fName + " " + lName,
			Subtitle:    subtitle,
			Description: "Patient",
			URL:         "/patients/" + id,
			Metadata:    map[string]any{},
		})
	}
	return results, rows.Err()
}
